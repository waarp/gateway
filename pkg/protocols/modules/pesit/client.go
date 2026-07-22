package pesit

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type clientService struct {
	db       *database.DB
	dbClient *model.Client
	state    utils.State
	logger   *log.Logger
	dialer   *protoutils.TraceDialer

	conf *ClientConfigTLS
}

func newClient(db *database.DB, cli *model.Client) *clientService {
	return &clientService{db: db, dbClient: cli}
}

func (c *clientService) Name() string { return c.dbClient.Name }

func (c *clientService) reportError(err error) {
	if err == nil {
		return
	}

	c.logger.Error(err.Error())
	c.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(c.dbClient.Name, err)
}

func (c *clientService) Start() (retErr error) {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	defer c.reportError(retErr)

	if err := c.db.Get(c.dbClient, "id=?", c.dbClient.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve client from database: %w", err)
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	c.logger.Info("Starting PeSIT client...")

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

	c.state.Set(utils.StateRunning, "")
	c.logger.Info("PeSIT client started successfully")

	return nil
}

func (c *clientService) Stop(ctx context.Context) (retErr error) {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	defer c.reportError(retErr)
	c.logger.Info("Stopping PeSIT client...")

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		return fmt.Errorf("failed to stop the PeSIT client: %w", err)
	}

	c.state.Set(utils.StateOffline, "")
	c.logger.Info("PeSIT client stopped successfully")

	return nil
}

func (c *clientService) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *clientService) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return c.initTransfer(pip)
}

func (c *clientService) initTransfer(pip *pipeline.Pipeline) (*clientTransfer, *pipeline.Error) {
	var pesitID uint32

	if pip.TransCtx.Rule.IsSend || pip.TransCtx.Transfer.Step > types.StepSetup {
		pesitID64, convErr := strconv.ParseUint(pip.TransCtx.Transfer.RemoteTransferID, 10, 32)
		if convErr != nil {
			return nil, pipeline.NewErrorWith(convErr, types.TeInternal, "failed to parse PeSIT transfer ID")
		}

		pesitID = uint32(pesitID64)
	}

	return &clientTransfer{
		isTLS:      c.dbClient.Protocol == PesitTLS,
		pip:        pip,
		clientConf: c.conf,
		dialer:     c.dialer,
		pesitID:    pesitID,
	}, nil
}
