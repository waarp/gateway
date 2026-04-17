package webdav

import (
	"context"
	"fmt"

	"github.com/studio-b12/gowebdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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

type client struct {
	db     *database.DB
	logger *log.Logger
	state  utils.State
	agent  *model.Client

	transporter httptransport.Transporter
}

func NewClient(db *database.DB, dbClient *model.Client) services.Client {
	return &client{
		db:    db,
		agent: dbClient,
	}
}

func (c *client) Name() string { return c.agent.Name }

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.agent.Name)
	if err := c.start(); err != nil {
		c.state.Set(utils.StateError, err.Error())
		c.logger.Errorf("failed to start the WebDAV client: %v", err)

		return err
	}

	c.state.Set(utils.StateRunning, "")
	c.logger.Info("WebDAV client started successfully")

	return nil
}

func (c *client) start() error {
	var err error
	if c.transporter, err = httptransport.NewTransport(c.agent.Protocol == WebdavTLS,
		c.agent.LocalAddress.String()); err != nil {
		c.logger.Errorf("Failed to initialize the WebDAV client's transport: %v", err)

		return fmt.Errorf("failed to initialize the WebDAV client's transport: %w", err)
	}

	return nil
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, c.agent.ID); err != nil {
		c.logger.Errorf("Failed to interrupt WebDAV client's running transfers: %v", err)
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.agent.Name, err)

		return fmt.Errorf("failed to stop the WebDAV client's running transfers: %w", err)
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	host := conf.GetRealAddress(pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(pip.TransCtx.RemoteAgent.Address.Port))
	login := pip.TransCtx.RemoteAccount.Login
	pswd := ""

	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			pswd = cred.Value
			break
		}
	}

	scheme := "http://"
	if pip.TransCtx.RemoteAgent.Protocol == WebdavTLS {
		scheme = "https://"
	}

	transport, err := c.transporter.Get(pip)
	if err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal, "failed to initialize WebDAV transport")
	}

	wdClient := gowebdav.NewClient(scheme+host, login, pswd)
	wdClient.SetTransport(transport)

	return &clientTransfer{
		client:  wdClient,
		pip:     pip,
		errChan: protoutils.NewErrChan(),
	}, nil
}
