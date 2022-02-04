package r66

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"strings"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
)

//nolint:gochecknoinits // init is used by design
func init() {
	pipeline.ClientConstructors["r66"] = NewClient
}

type client struct {
	pip *pipeline.Pipeline

	conf      config.R66ProtoConfig
	tlsConfig *tls.Config

	ctx    context.Context //nolint:containedctx //FIXME move the context to a function parameter
	cancel func()
	ses    *r66.Session
}

// NewClient creates and returns a new r66 client using the given transfer context.
func NewClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return newClient(pip)
}

func newClient(pip *pipeline.Pipeline) (*client, *types.TransferError) {
	var protoConfig config.R66ProtoConfig
	if err := json.Unmarshal(pip.TransCtx.RemoteAgent.ProtoConfig, &protoConfig); err != nil {
		pip.Logger.Errorf("Failed to parse R66 partner proto config: %s", err)

		return nil, types.NewTransferError(types.TeInternal, "failed to parse R66 partner proto config")
	}

	var tlsConf *tls.Config

	if protoConfig.IsTLS {
		var err error

		tlsConf, err = internal.MakeClientTLSConfig(pip.TransCtx)
		if err != nil {
			pip.Logger.Errorf("Failed to parse R66 TLS config: %s", err)

			return nil, types.NewTransferError(types.TeInternal, "invalid R66 TLS config")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &client{
		pip:       pip,
		conf:      protoConfig,
		tlsConfig: tlsConf,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Request opens a connection to the remote partner, creates a new authenticated
// session, and sends the transfer request.
func (c *client) Request() *types.TransferError {
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
func (c *client) BeginPreTasks() *types.TransferError { return nil }

// EndPreTasks sends/receives updated transfer info to/from the remote partner.
func (c *client) EndPreTasks() *types.TransferError {
	if c.pip.TransCtx.Rule.IsSend {
		info := &r66.UpdateInfo{
			Filename: strings.TrimPrefix(c.pip.TransCtx.Transfer.RemotePath, "/"),
			FileSize: c.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}

		if err := c.ses.SendUpdateRequest(info); err != nil {
			c.pip.Logger.Errorf("Failed to send transfer info: %s", err)

			return internal.FromR66Error(err, c.pip)
		}

		return nil
	}

	info, err := c.ses.RecvUpdateRequest()
	if err != nil {
		c.pip.Logger.Errorf("Failed to receive transfer info: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	return internal.UpdateInfo(info, c.pip)
}

// Data copies data between the given data stream and the remote partner.
func (c *client) Data(dataStream pipeline.DataStream) *types.TransferError {
	if c.pip.TransCtx.Rule.IsSend {
		_, err := c.ses.Send(dataStream, c.makeHash)
		if err != nil {
			c.pip.Logger.Errorf("Failed to send transfer file: %s", err)

			return internal.FromR66Error(err, c.pip)
		}

		return nil
	}

	eot, err := c.ses.Recv(dataStream)
	if err != nil {
		c.pip.Logger.Errorf("Failed to receive transfer file: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	if c.conf.NoFinalHash {
		return nil
	}

	hash, hErr := internal.MakeHash(c.ctx, c.pip.Logger, c.pip.TransCtx.Transfer.LocalPath)
	if hErr != nil {
		return hErr
	}

	if !bytes.Equal(eot.Hash, hash) {
		c.pip.Logger.Errorf("File hash does not match expected value")

		return types.NewTransferError(types.TeIntegrity, "invalid file hash")
	}

	return nil
}

// EndTransfer send a transfer end message, and then closes the session.
func (c *client) EndTransfer() *types.TransferError {
	defer c.cancel()
	defer clientConns.Done(c.pip.TransCtx.RemoteAgent.Address)
	defer func() {
		if c.ses != nil {
			c.ses.Close()
		}
	}()

	c.pip.Logger.Debug("Ending transfert with remote partner")

	if err := c.ses.EndRequest(); err != nil {
		c.pip.Logger.Errorf("Failed to end transfer request: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}

// SendError sends the given error to the remote partner and then closes the
// session.
func (c *client) SendError(err *types.TransferError) {
	c.pip.Logger.Debugf("Sending error '%s' to remote partner", err)

	defer c.cancel()
	defer clientConns.Done(c.pip.TransCtx.RemoteAgent.Address)

	if c.ses == nil {
		return
	}

	defer c.ses.Close()

	if sErr := c.ses.SendError(internal.ToR66Error(err)); sErr != nil {
		c.pip.Logger.Errorf("Failed to send error to remote partner: %s", sErr)
	}
}

// Pause sends a pause message to the remote partner and then closes the
// session.
func (c *client) Pause() *types.TransferError {
	defer c.cancel()
	defer func() {
		clientConns.Done(c.pip.TransCtx.RemoteAgent.Address)
	}()

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Stop(); err != nil {
		c.pip.Logger.Warningf("Failed send pause signal to remote host: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}

// Cancel sends a cancel message to the remote partner and then closes the
// session.
func (c *client) Cancel(context.Context) *types.TransferError {
	defer c.cancel()
	defer func() {
		clientConns.Done(c.pip.TransCtx.RemoteAgent.Address)
	}()

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Cancel(); err != nil {
		c.pip.Logger.Warningf("Failed send cancel signal to remote host: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	return nil
}
