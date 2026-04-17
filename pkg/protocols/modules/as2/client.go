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
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errClientShuttingDown = pipeline.NewError(types.TeShuttingDown, "AS2 client is shutting down")

type client struct {
	db       *database.DB
	dbClient *model.Client

	state       utils.State
	logger      *log.Logger
	protoConfig *clientProtoConfigTLS

	transporter httptransport.Transporter
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

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	c.logger.Info("Starting AS2 client...")

	if err := c.start(); err != nil {
		c.logger.Errorf("Failed to start AS2 client: %v", err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.logger.Info("AS2 client started successfully")
	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) start() error {
	if err := utils.JSONConvert(c.dbClient.ProtoConfig, &c.protoConfig); err != nil {
		return fmt.Errorf("invalid client config: %w", err)
	}

	var err error
	if c.transporter, err = httptransport.NewTransport(c.dbClient.Protocol == AS2TLS,
		c.dbClient.LocalAddress.String()); err != nil {
		return fmt.Errorf("failed to initialize the AS2 client's transport: %w", err)
	}

	c.ctx, c.cancel = context.WithCancelCause(context.Background())

	return nil
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	c.logger.Info("Stopping AS2 client...")

	if err := c.stop(ctx); err != nil {
		c.logger.Errorf("Failed to stop AS2 client: %v", err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.logger.Info("AS2 client stopped successfully")
	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) stop(ctx context.Context) error {
	defer c.cancel(errClientShuttingDown)

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		return fmt.Errorf("failed to stop running transfers: %w", err)
	}

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

func (c *client) getTransport(pip *pipeline.Pipeline) (http.RoundTripper, *pipeline.Error) {
	transport, err := c.transporter.Get(pip)
	if err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal,
			"failed to initialize the AS2 client's transport")
	}

	return newHTTPTransport(transport, pip), nil
}

func (c *client) listenAsync(pip *pipeline.Pipeline) *pipeline.Error {
	if _, err := c.asyncLists.Connect(pip); err != nil {
		return pipeline.NewErrorWith(err, types.TeConnection, "failed to start async MDN listener")
	}

	return nil
}

type httpTransport struct {
	rt          http.RoundTripper
	login, pswd string
}

func newHTTPTransport(rt http.RoundTripper, pip *pipeline.Pipeline) http.RoundTripper {
	login := pip.TransCtx.RemoteAccount.Login
	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			return &httpTransport{rt, login, cred.Value}
		}
	}

	return rt
}

func (h *httpTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if h.login != "" && h.pswd != "" {
		r.SetBasicAuth(h.login, h.pswd)
	}

	//nolint:wrapcheck //no need to wrap here
	return h.rt.RoundTrip(r)
}
