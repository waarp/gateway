package webdav

import (
	"context"
	"fmt"
	"net/http"

	"github.com/studio-b12/gowebdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
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

	transporter *httptransport.Transporter
}

func newClient(db *database.DB, dbClient *model.Client) *client {
	return &client{
		db:    db,
		agent: dbClient,
	}
}

func (c *client) Name() string { return c.agent.Name }

func (c *client) reportError(err error) {
	if err == nil {
		return
	}

	c.logger.Error(err.Error())
	c.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(c.agent.Name, err)
}

func (c *client) Start() (retErr error) {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.agent.Name)
	defer c.reportError(retErr)

	if err := c.db.Get(c.agent, "id=?", c.agent.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve client from database: %w", err)
	}

	c.logger = logging.NewLogger(c.agent.Name)
	c.logger.Info("Starting WebDAV client...")

	var err error
	if c.transporter, err = httptransport.NewTransporter(c.agent.Protocol == WebdavTLS,
		c.agent.LocalAddress.String(), c.db.Config.Overrides); err != nil {
		return fmt.Errorf("failed to initialize the WebDAV client's transport: %w", err)
	}

	c.state.Set(utils.StateRunning, "")
	c.logger.Info("WebDAV client started successfully")

	return nil
}

func (c *client) Stop(ctx context.Context) (retErr error) {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	c.logger.Info("Stopping WebDAV client...")
	defer c.reportError(retErr)
	defer c.transporter.Close()

	if err := pipeline.List.StopAllFromClient(ctx, c.agent.ID); err != nil {
		return fmt.Errorf("failed to stop the WebDAV client's running transfers: %w", err)
	}

	c.logger.Info("WebDAV client stopped successfully")
	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	transport := c.transporter.Connect(pip)
	wdClient := getClient(pip.TransCtx, transport, pip.DB.Config.Overrides)

	return &clientTransfer{
		client:  wdClient,
		pip:     pip,
		errChan: protoutils.NewErrChan(),
	}, nil
}

func GetClient(logger *log.Logger, ctx *model.TransferContext, overrides *conf.ConfigOverride,
) (*gowebdav.Client, error) {
	isTLS := ctx.Client.Protocol == WebdavTLS
	transporter, err := httptransport.NewTransporter(isTLS, ctx.Client.LocalAddress.String(), overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the WebDAV client's transport: %w", err)
	}

	transport := transporter.Connect(&pipeline.Pipeline{
		TransCtx: ctx,
		Logger:   logger,
	})

	return getClient(ctx, transport, overrides), nil
}

func getClient(ctx *model.TransferContext, transport http.RoundTripper,
	overrides *conf.ConfigOverride,
) *gowebdav.Client {
	host := overrides.GetRealAddress(ctx.RemoteAgent.Address.Host,
		utils.FormatUint(ctx.RemoteAgent.Address.Port))
	login := ctx.RemoteAccount.Login
	pswd := ""

	for _, cred := range ctx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			pswd = cred.Value
			break
		}
	}

	scheme := "http://"
	if ctx.RemoteAgent.Protocol == WebdavTLS {
		scheme = "https://"
	}

	wdClient := gowebdav.NewClient(scheme+host, login, pswd)
	wdClient.SetTransport(transport)

	return wdClient
}
