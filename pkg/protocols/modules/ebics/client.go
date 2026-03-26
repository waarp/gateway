package ebics

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Client struct {
	db     *database.DB
	logger *log.Logger
	client *model.Client
	config *clientConfig
	state  utils.State
}

func NewClient(db *database.DB, dbClient *model.Client) *Client {
	return &Client{db: db, client: dbClient}
}

func (c *Client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.client.Name)
	cfg := defaultClientConfig()
	if err := utils.JSONConvert(c.client.ProtoConfig, cfg); err != nil {
		err = wrapConfigError(err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	if err := cfg.ValidClient(); err != nil {
		err = wrapConfigError(err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.config = cfg
	c.state.Set(utils.StateRunning, "")
	c.logger.Info("EBICS client bootstrap completed")

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}
	_ = ctx
	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *Client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *Client) InitTransfer(_ *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return nil, pipeline.NewErrorWith(
		types.TeUnimplemented,
		"EBICS transfer bootstrap is not implemented yet",
		fmt.Errorf("%w", ErrNotImplemented),
	)
}
