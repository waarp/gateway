package as2

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/lib/log/v2"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httptransport"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errClientShuttingDown = pipeline.NewError(types.TeShuttingDown, "AS2 client is shutting down")

type client struct {
	db       *database.DB
	dbClient *model.Client

	state       utils.State
	logger      *log.Logger
	protoConfig *clientProtoConfigTLS

	transporter *httptransport.Transporter
	asyncStore  *asyncStore
	asyncLists  *protoutils.ConnPool[net.Listener]

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func NewClient(db *database.DB, dbClient *model.Client) protocol.Client {
	store := newAsyncStore()
	connPool := protoutils.NewConnPool(nil, store.asyncListen)
	connPool.SetGracePeriod(0)

	return &client{
		db:         db,
		dbClient:   dbClient,
		asyncLists: connPool,
		asyncStore: store,
	}
}

func (c *client) Name() string { return c.dbClient.Name }

func (c *client) reportError(err error) {
	if err == nil {
		return
	}

	c.logger.Error(err.Error())
	c.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(c.dbClient.Name, err)
}

func (c *client) Start() (retErr error) {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	defer c.reportError(retErr)

	if err := c.db.Get(c.dbClient, "id=?", c.dbClient.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve client from database: %w", err)
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	c.logger.Info("Starting AS2 client...")

	if err := utils.JSONConvert(c.dbClient.ProtoConfig, &c.protoConfig); err != nil {
		return fmt.Errorf("invalid client config: %w", err)
	}

	var err error
	if c.transporter, err = httptransport.NewTransporter(c.dbClient.Protocol == AS2TLS,
		c.dbClient.LocalAddress.String(), c.db.Config.Overrides); err != nil {
		return fmt.Errorf("failed to initialize the AS2 client's transport: %w", err)
	}

	c.ctx, c.cancel = context.WithCancelCause(context.Background())

	c.logger.Info("AS2 client started successfully")
	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) Stop(ctx context.Context) (retErr error) {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	c.logger.Info("Stopping AS2 client...")
	defer c.reportError(retErr)
	defer c.cancel(errClientShuttingDown)
	defer c.transporter.Close()

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		return fmt.Errorf("failed to stop running transfers: %w", err)
	}

	c.logger.Info("AS2 client stopped successfully")
	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	var partConf partnerProtoConfigTLS
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partConf); err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal, "invalid partner config")
	}

	cliTrans, cliErr := c.newClientTransfer(c.ctx, pip, partConf)
	if cliErr != nil {
		return nil, cliErr
	}

	if partConf.HandleAsyncMDN {
		if err := c.listenAsync(pip); err != nil {
			return nil, err
		}

		msgID := pip.TransCtx.Transfer.RemoteTransferID
		c.asyncStore.m.Store(msgID, &cliTrans.asyncChan)
		cliTrans.asyncChan.Init()
		cliTrans.done = func() {
			c.asyncLists.CloseConn(pip)
			c.asyncStore.m.Delete(msgID)
		}
	}

	return cliTrans, nil
}

func (c *client) getTransport(pip *pipeline.Pipeline) http.RoundTripper {
	transport := c.transporter.Connect(pip)

	return newAs2Transport(transport, pip)
}

func (c *client) listenAsync(pip *pipeline.Pipeline) *pipeline.Error {
	if _, err := c.asyncLists.Connect(pip); err != nil {
		return pipeline.NewErrorWith(err, types.TeConnection, "failed to start async MDN listener")
	}

	return nil
}

type as2Transport struct {
	rt          http.RoundTripper
	login, pswd string
}

func newAs2Transport(rt http.RoundTripper, pip *pipeline.Pipeline) http.RoundTripper {
	login := pip.TransCtx.RemoteAccount.Login
	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			return &as2Transport{rt, login, cred.Value}
		}
	}

	return rt
}

func (t *as2Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.login != "" && t.pswd != "" {
		r.SetBasicAuth(t.login, t.pswd)
	}

	//nolint:wrapcheck //no need to wrap here
	return t.rt.RoundTrip(r)
}
