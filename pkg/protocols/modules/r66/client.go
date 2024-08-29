package r66

import (
	"context"
	"crypto/tls"
	"fmt"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Client struct {
	db               *database.DB
	cli              *model.Client
	disableConnGrace bool

	logger       *log.Logger
	clientConfig *clientConfig
	conns        *internal.ConnPool
	state        utils.State
}

func (c *Client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := c.start(); err != nil {
		c.logger.Error("Failed to start R66 client: %v", err)
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.cli.Name, err)

		return err
	}

	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *Client) start() error {
	c.logger = logging.NewLogger(c.cli.Name)

	var conf clientConfig
	if err := utils.JSONConvert(c.cli.ProtoConfig, &conf); err != nil {
		return fmt.Errorf("failed to parse the R66 client's config: %w", err)
	}

	connPool, err := internal.NewConnPool(c.cli)
	if err != nil {
		return fmt.Errorf("failed to initialize the R66 client's connection pool: %w", err)
	}

	if c.disableConnGrace {
		connPool.SetGracePeriod(0)
	}

	c.clientConfig = &conf
	c.conns = connPool

	return nil
}

func (c *Client) State() (utils.StateCode, string) { return c.state.Get() }

func (c *Client) Stop(ctx context.Context) error {
	defer c.conns.ForceClose()

	if err := pipeline.List.StopAllFromClient(ctx, c.cli.ID); err != nil {
		c.state.Set(utils.StateError, fmt.Sprintf("failed to stop transfers: %v", err))
		snmp.ReportServiceFailure(c.cli.Name, err)

		return fmt.Errorf("failed to stop transfers: %w", err)
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *Client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	trans, err := c.initTransfer(pip)
	if err != nil {
		return nil, err
	}

	return trans, nil
}

//nolint:funlen //can't easily be split
func (c *Client) initTransfer(pip *pipeline.Pipeline) (*transferClient, *pipeline.Error) {
	var partConf partnerConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partConf); err != nil {
		pip.Logger.Error("Failed to parse R66 partner proto config: %v", err)

		return nil, pipeline.NewErrorWith(types.TeInternal,
			"failed to parse R66 partner proto config", err)
	}

	var tlsConf *tls.Config

	if c.cli.Protocol == R66TLS {
		var err error

		tlsConf, err = makeClientTLSConfig(pip)
		if err != nil {
			pip.Logger.Error("Failed to parse R66 TLS config: %v", err)

			return nil, pipeline.NewErrorWith(types.TeInternal, "invalid R66 TLS config", err)
		}
	}

	var blockSize uint32 = 65536
	if c.clientConfig.BlockSize != 0 {
		blockSize = c.clientConfig.BlockSize
		if partConf.BlockSize != 0 {
			blockSize = partConf.BlockSize
		}
	}

	noFinalHash := c.clientConfig.NoFinalHash
	if partConf.NoFinalHash != nil {
		noFinalHash = *partConf.CheckBlockHash
	}

	var finalHashAlgo string = internal.HashSHA256
	if c.clientConfig.FinalHashAlgo != "" {
		finalHashAlgo = c.clientConfig.FinalHashAlgo
		if partConf.FinalHashAlgo != "" {
			finalHashAlgo = partConf.FinalHashAlgo
		}
	}

	checkBlockHash := c.clientConfig.CheckBlockHash
	if partConf.CheckBlockHash != nil {
		checkBlockHash = *partConf.CheckBlockHash
	}

	partnerLogin := partConf.ServerLogin
	if partnerLogin == "" {
		partnerLogin = pip.TransCtx.RemoteAgent.Name
	}

	var partnerPassword string

	for _, cred := range pip.TransCtx.RemoteAgentCreds {
		if cred.Type == auth.Password {
			partnerPassword = cred.Value

			break
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &transferClient{
		conns:          c.conns,
		pip:            pip,
		ctx:            ctx,
		cancel:         cancel,
		blockSize:      blockSize,
		noFinalHash:    noFinalHash,
		finalHashAlgo:  finalHashAlgo,
		checkBlockHash: checkBlockHash,
		serverLogin:    partnerLogin,
		serverPassword: partnerPassword,
		tlsConfig:      tlsConf,
		ses:            nil,
	}, nil
}

func (c *Client) GetConnection(partner *model.RemoteAgent, account *model.RemoteAccount) (*r66.Client, error) {
	remoteAgentCreds, err1 := partner.GetCredentials(c.db)
	if err1 != nil {
		return nil, fmt.Errorf("failed to retrieve partner credentials: %w", err1)
	}

	remoteAccountCreds, err2 := account.GetCredentials(c.db)
	if err2 != nil {
		return nil, fmt.Errorf("failed to retrieve account credentials: %w", err2)
	}

	fakeTransCtx := &model.TransferContext{
		RemoteAgent:        partner,
		RemoteAccount:      account,
		RemoteAccountCreds: remoteAccountCreds,
		RemoteAgentCreds:   remoteAgentCreds,
	}
	fakePipeline := &pipeline.Pipeline{
		DB:       c.db,
		Logger:   c.logger,
		TransCtx: fakeTransCtx,
	}

	transferCli, cliErr := c.initTransfer(fakePipeline)
	if cliErr != nil {
		return nil, cliErr
	}

	conn, connErr := transferCli.connect()
	if connErr != nil {
		return nil, connErr
	}

	return conn, nil
}

func (c *Client) ReturnConnection(partner *model.RemoteAgent) {
	c.conns.Done(partner.Address.String())
}
