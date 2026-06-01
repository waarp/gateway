package pesit

import (
	"context"
	"fmt"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type client struct {
	dbClient *model.Client
	state    utils.State
	logger   *log.Logger
	isTLS    bool
	conns    *pesitConnPool

	conf *ClientConfigTLS
}

func newClient(cli *model.Client) *client {
	return &client{dbClient: cli}
}

func (c *client) Name() string { return c.dbClient.Name }

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	if err := c.start(); err != nil {
		c.state.Set(utils.StateError, err.Error())
		c.logger.Errorf("failed to start the PeSIT client: %v", err)

		return err
	}

	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) start() error {
	c.conf = &ClientConfigTLS{}
	if err := utils.JSONConvert(c.dbClient.ProtoConfig, c.conf); err != nil {
		return fmt.Errorf("invalid client config: %w", err)
	}

	c.isTLS = c.dbClient.Protocol == PesitTLS
	dialer := makePesitDialer(c.dbClient)
	c.conns = protoutils.NewConnPool[*pesitClientConn](dialer, c.dialPesitConn)
	// PeSIT connections are sequential (one transfer at a time), so we use
	// exclusive mode: concurrent transfers get their own connections instead
	// of sharing. Sequential reuse via the grace period still works.
	c.conns.SetExclusive(true)

	return nil
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := c.stop(ctx); err != nil {
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) stop(ctx context.Context) error {
	//nolint:errcheck //we don't care about the error here
	defer c.conns.Stop()

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		c.logger.Errorf("Failed to stop the PeSIT client: %v", err)

		return fmt.Errorf("failed to stop the PeSIT client: %w", err)
	}

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return c.initTransfer(pip)
}

func (c *client) initTransfer(pip *pipeline.Pipeline) (*clientTransfer, *pipeline.Error) {
	var pesitID uint32

	if pip.TransCtx.Rule.IsSend || pip.TransCtx.Transfer.Step > types.StepSetup {
		pesitID64, convErr := strconv.ParseUint(pip.TransCtx.Transfer.RemoteTransferID, 10, 32)
		if convErr != nil {
			return nil, pipeline.NewErrorWith(types.TeInternal, "failed to parse PeSIT transfer ID", convErr)
		}

		pesitID = uint32(pesitID64)
	}

	return &clientTransfer{
		isTLS:      c.isTLS,
		pip:        pip,
		clientConf: c.conf,
		conns:      c.conns,
		pesitID:    pesitID,
	}, nil
}
