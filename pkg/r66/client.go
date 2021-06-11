package r66

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
)

func init() {
	pipeline.ClientConstructors["r66"] = NewClient
}

type client struct {
	pip *pipeline.Pipeline

	conf      config.R66ProtoConfig
	tlsConfig *tls.Config

	ses *r66.Session
}

// NewClient creates and returns a new r66 client using the given transfer context.
func NewClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	var conf config.R66ProtoConfig
	if err := json.Unmarshal(pip.TransCtx.RemoteAgent.ProtoConfig, &conf); err != nil {
		pip.Logger.Errorf("Failed to parse R66 partner proto config: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to parse R66 partner proto config")
	}

	var tlsConf *tls.Config
	if conf.IsTLS {
		var err error
		tlsConf, err = internal.MakeClientTLSConfig(pip.TransCtx)
		if err != nil {
			pip.Logger.Errorf("Failed to parse R66 TLS config: %s", err)
			return nil, types.NewTransferError(types.TeInternal, "invalid R66 TLS config")
		}
	}

	return &client{
		pip:       pip,
		conf:      conf,
		tlsConfig: tlsConf,
	}, nil
}

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
	if err := c.request(); err != nil {
		return err
	}

	return nil
}

func (c *client) BeginPreTasks() *types.TransferError { return nil }

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

	hash, hErr := internal.MakeHash(c.pip.Logger, c.pip.TransCtx.Transfer.LocalPath)
	if hErr != nil {
		return hErr
	}
	if !bytes.Equal(eot.Hash, hash) {
		c.pip.Logger.Errorf("File hash does not match expected value")
		return types.NewTransferError(types.TeIntegrity, "invalid file hash")
	}

	return nil
}

func (c *client) EndTransfer() *types.TransferError {
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

func (c *client) SendError(err *types.TransferError) {
	defer clientConns.Done(c.pip.TransCtx.RemoteAgent.Address)
	defer func() {
		if c.ses != nil {
			c.ses.Close()
		}
	}()
	c.pip.Logger.Debugf("Sending error '%s' to remote partner", err)
	if sErr := c.ses.SendError(internal.ToR66Error(err)); sErr != nil {
		c.pip.Logger.Errorf("Failed to send error to remote partner: %s", sErr)
	}
}

func (c *client) Pause() *types.TransferError {
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

func (c *client) Cancel() *types.TransferError {
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
