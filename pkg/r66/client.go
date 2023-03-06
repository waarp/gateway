package r66

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type client struct {
	cli          *model.Client
	clientConfig *config.R66ClientProtoConfig
	transfers    *service.TransferMap
	conns        *internal.ConnPool
	state        state.State
}

// NewClient creates and returns a new r66 client using the given transfer context.
func NewClient(dbClient *model.Client) (pipeline.Client, error) {
	var clientConfig config.R66ClientProtoConfig
	if err := utils.JSONConvert(dbClient.ProtoConfig, &clientConfig); err != nil {
		return nil, fmt.Errorf("failed to parse the R66 client's config: %w", err)
	}

	connPool, err := internal.NewConnPool(dbClient, &clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the R66 client's connection pool: %w", err)
	}

	return &client{
		cli:          dbClient,
		clientConfig: &clientConfig,
		transfers:    service.NewTransferMap(),
		conns:        connPool,
	}, nil
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	var partnerConfig config.R66PartnerProtoConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partnerConfig); err != nil {
		pip.Logger.Error("Failed to parse R66 partner proto config: %v", err)

		return nil, types.NewTransferError(types.TeInternal, "failed to parse R66 partner proto config")
	}

	var tlsConf *tls.Config

	if c.cli.Protocol == ProtocolR66TLS {
		var err error

		tlsConf, err = internal.MakeClientTLSConfig(pip)
		if err != nil {
			pip.Logger.Error("Failed to parse R66 TLS config: %v", err)

			return nil, types.NewTransferError(types.TeInternal, "invalid R66 TLS config")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	var blockSize uint32 = 65536
	if c.clientConfig.BlockSize != 0 {
		blockSize = c.clientConfig.BlockSize
		if partnerConfig.BlockSize != 0 {
			blockSize = partnerConfig.BlockSize
		}
	}

	noFinalHash := c.clientConfig.NoFinalHash
	if partnerConfig.NoFinalHash != nil {
		noFinalHash = *partnerConfig.CheckBlockHash
	}

	checkBlockHash := c.clientConfig.CheckBlockHash
	if partnerConfig.CheckBlockHash != nil {
		checkBlockHash = *partnerConfig.CheckBlockHash
	}

	return &transferClient{
		conns:          c.conns,
		pip:            pip,
		ctx:            ctx,
		cancel:         cancel,
		blockSize:      blockSize,
		noFinalHash:    noFinalHash,
		checkBlockHash: checkBlockHash,
		serverLogin:    partnerConfig.ServerLogin,
		serverPassword: partnerConfig.ServerPassword,
		tlsConfig:      tlsConf,
		ses:            nil,
	}, nil
}

func (c *client) Start() error {
	c.state.Set(state.Running, "")

	return nil
}

func (c *client) State() *state.State { return &c.state }

func (c *client) Stop(ctx context.Context) error {
	if err := c.transfers.InterruptAll(ctx); err != nil {
		return fmt.Errorf("failed to interrupt the running transfers: %w", err)
	}

	c.conns.ForceClose()

	c.state.Set(state.Offline, "")

	return nil
}

func (c *client) ManageTransfers() *service.TransferMap {
	return c.transfers
}

type transferClient struct {
	conns *internal.ConnPool
	pip   *pipeline.Pipeline

	ctx    context.Context //nolint:containedctx //FIXME move the context to a function parameter
	cancel func()

	blockSize                   uint32
	noFinalHash, checkBlockHash bool
	serverLogin, serverPassword string

	tlsConfig *tls.Config
	ses       *r66.Session
}

// Request opens a connection to the remote partner, creates a new authenticated
// session, and sends the transfer request.
func (c *transferClient) Request() *types.TransferError {
	// CONNECTION
	if err := c.connect(); err != nil {
		return err
	}

	// AUTHENTICATION
	if err := c.authenticate(); err != nil {
		return err
	}

	// REQUEST
	return c.request()
}

// BeginPreTasks does nothing (needed to implement PreTaskHandler).
func (c *transferClient) BeginPreTasks() *types.TransferError { return nil }

// EndPreTasks sends/receives updated transfer info to/from the remote partner.
func (c *transferClient) EndPreTasks() *types.TransferError {
	if c.pip.TransCtx.Rule.IsSend {
		outInfo := &r66.UpdateInfo{
			Filename: c.pip.TransCtx.Transfer.RemotePath,
			FileSize: c.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}

		if err := internal.MakeTransferInfo(c.pip.Logger, c.pip.TransCtx,
			outInfo.FileInfo); err != nil {
			return err
		}

		/*
			if err := internal.MakeFileInfo(c.pip, &outInfo.FileInfo.SystemData); err != nil {
				return err
			}
		*/

		if err := c.ses.SendUpdateRequest(outInfo); err != nil {
			c.pip.Logger.Error("Failed to send transfer info: %v", err)

			return internal.FromR66Error(err, c.pip)
		}

		return nil
	}

	inInfo, err := c.ses.RecvUpdateRequest()
	if err != nil {
		c.pip.Logger.Error("Failed to receive transfer info: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	return internal.UpdateFileInfo(inInfo, c.pip)
}

// Data copies data between the given data stream and the remote partner.
func (c *transferClient) Data(dataStream pipeline.DataStream) *types.TransferError {
	stream := &clientStream{stream: dataStream}

	if c.pip.TransCtx.Rule.IsSend {
		_, err := c.ses.Send(stream, c.makeHash)
		if err != nil {
			c.pip.Logger.Error("Failed to send transfer file: %v", err)

			return internal.FromR66Error(err, c.pip)
		}

		return nil
	}

	eot, err := c.ses.Recv(stream)
	if err != nil {
		c.pip.Logger.Error("Failed to receive transfer file: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	if c.noFinalHash {
		return nil
	}

	hash, hErr := internal.MakeHash(c.ctx, c.pip.TransCtx.FS, c.pip.Logger,
		&c.pip.TransCtx.Transfer.LocalPath)
	if hErr != nil {
		return hErr
	}

	if !bytes.Equal(eot.Hash, hash) {
		c.pip.Logger.Error("File hash does not match expected value")

		return types.NewTransferError(types.TeIntegrity, "invalid file hash")
	}

	return nil
}

// EndTransfer send a transfer end message, and then closes the session.
func (c *transferClient) EndTransfer() *types.TransferError {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address)

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	c.pip.Logger.Debug("Ending transfert with remote partner")

	if err := c.ses.EndRequest(); err != nil {
		c.pip.Logger.Error("Failed to end transfer request: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}

// SendError sends the given error to the remote partner and then closes the
// session.
func (c *transferClient) SendError(err *types.TransferError) {
	c.pip.Logger.Debug("Sending error '%v' to remote partner", err)

	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address)

	if c.ses == nil {
		return
	}

	defer c.ses.Close()

	if sErr := c.ses.SendError(internal.ToR66Error(err)); sErr != nil {
		c.pip.Logger.Error("Failed to send error to remote partner: %v", sErr)
	}
}

// Pause sends a pause message to the remote partner and then closes the
// session.
func (c *transferClient) Pause() *types.TransferError {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address)

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Stop(); err != nil {
		c.pip.Logger.Warning("Failed send pause signal to remote host: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}

// Cancel sends a cancel message to the remote partner and then closes the
// session.
func (c *transferClient) Cancel(context.Context) *types.TransferError {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address)

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Cancel(); err != nil {
		c.pip.Logger.Warning("Failed send cancel signal to remote host: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}
