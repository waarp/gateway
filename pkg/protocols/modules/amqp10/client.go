package amqp10

import (
	"context"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type client struct {
	dbClient *model.Client
	state    utils.State
	logger   *log.Logger
	conf     *clientConfig
	dialer   amqpDialer
}

func newClient(dbClient *model.Client) *client {
	return &client{
		dbClient: dbClient,
		dialer:   defaultDialer{},
	}
}

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	if err := c.start(); err != nil {
		c.state.Set(utils.StateError, err.Error())
		c.logger.Errorf("Failed to start the AMQP 1.0 client: %v", err)
		snmp.ReportServiceFailure(c.dbClient.Name, err)

		return err
	}

	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) start() error {
	c.conf = defaultClientConfig()
	if err := utils.JSONConvert(c.dbClient.ProtoConfig, c.conf); err != nil {
		return fmt.Errorf("invalid AMQP 1.0 client config: %w", err)
	}

	return c.conf.ValidClient()
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.dbClient.Name, err)

		return fmt.Errorf("failed to stop the AMQP 1.0 client: %w", err)
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) State() (utils.StateCode, string) { return c.state.Get() }

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	partConf := defaultPartnerConfig()
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, partConf); err != nil {
		return nil, pipeline.NewErrorWith(types.TeConnection, "invalid AMQP 1.0 partner config", err)
	}
	if err := partConf.ValidPartner(); err != nil {
		return nil, pipeline.NewErrorWith(types.TeConnection, "invalid AMQP 1.0 partner config", err)
	}

	return &transferClient{
		pip:     pip,
		conf:    *c.conf,
		partner: *partConf,
		dialer:  c.dialer,
		timeout: time.Duration(c.conf.ConsumeTimeout) * time.Second,
	}, nil
}
