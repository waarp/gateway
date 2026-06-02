package pesit

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/puzpuzpuz/xsync/v4"

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
	dbClient   *model.Client
	state      utils.State
	logger     *log.Logger
	dialer     *protoutils.TraceDialer
	semaphores *xsync.Map[int64, chan struct{}]

	conf *ClientConfigTLS
}

func newClient(cli *model.Client) *client {
	return &client{
		dbClient:   cli,
		semaphores: xsync.NewMap[int64, chan struct{}](),
	}
}

// acquireConnSlot blocks until a connection slot is available for the
// given partner. Returns a release function to call when done.
// If maxConnections is 0, no limit is applied.
func (c *client) acquireConnSlot(partnerID int64, maxConnections uint16) func() {
	if maxConnections == 0 {
		return func() {} // no-op
	}

	sem, _ := c.semaphores.LoadOrCompute(partnerID, func() (chan struct{}, bool) {
		return make(chan struct{}, maxConnections), false
	})

	sem <- struct{}{} // blocks if full

	return func() { <-sem }
}

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

	c.dialer = &protoutils.TraceDialer{Dialer: &net.Dialer{}}

	if c.dbClient.LocalAddress.IsSet() {
		var err error
		if c.dialer.LocalAddr, err = net.ResolveTCPAddr("tcp", c.dbClient.LocalAddress.String()); err != nil {
			return fmt.Errorf("failed to parse the PeSIT client's local address: %w", err)
		}
	}

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
		isTLS:          c.dbClient.Protocol == PesitTLS,
		pip:            pip,
		clientConf:     c.conf,
		dialer:         c.dialer,
		pesitID:        pesitID,
		acquireConnFn:  c.acquireConnSlot,
	}, nil
}
