package r66

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type transferClient struct {
	conns *internal.ConnPool
	pip   *pipeline.Pipeline

	ctx    context.Context
	cancel func()

	blockSize                   uint32
	noFinalHash, checkBlockHash bool
	serverLogin, serverPassword string

	tlsConfig *tls.Config
	ses       *r66.Session
}

// Request opens a connection to the remote partner, creates a new authenticated
// session, and sends the transfer request.
func (c *transferClient) Request() *pipeline.Error {
	// CONNECTION
	if err := c.connect(); err != nil {
		return c.wrapAndSendError(err)
	}

	// AUTHENTICATION
	if err := c.authenticate(); err != nil {
		return c.wrapAndSendError(err)
	}

	// REQUEST
	if err := c.sendRequest(); err != nil {
		return c.wrapAndSendError(err)
	}

	return nil
}

// BeginPreTasks does nothing (needed to implement PreTaskHandler).
func (c *transferClient) BeginPreTasks() *pipeline.Error { return nil }

// EndPreTasks sends/receives updated transfer info to/from the remote partner.
func (c *transferClient) EndPreTasks() *pipeline.Error {
	if c.pip.TransCtx.Rule.IsSend {
		outInfo := &r66.UpdateInfo{
			Filename: c.pip.TransCtx.Transfer.RemotePath,
			FileSize: c.pip.TransCtx.Transfer.Filesize,
			FileInfo: &r66.TransferData{},
		}

		if err := internal.MakeTransferInfo(c.pip.Logger, c.pip.TransCtx,
			outInfo.FileInfo); err != nil {
			return c.wrapAndSendError(err)
		}

		if err := c.ses.SendUpdateRequest(outInfo); err != nil {
			c.pip.Logger.Error("Failed to send transfer info: %v", err)

			return c.wrapAndSendError(err)
		}

		return nil
	}

	inInfo, reqErr := c.ses.RecvUpdateRequest()
	if reqErr != nil {
		c.pip.Logger.Error("Failed to receive transfer info: %v", reqErr)

		return c.wrapAndSendError(reqErr)
	}

	if err := internal.UpdateFileInfo(inInfo, c.pip); err != nil {
		return c.wrapAndSendError(err)
	}

	return nil
}

func (c *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	if _, err := c.ses.Send(clientReader{r: file}, c.makeHash); err != nil {
		c.ses = nil
		c.pip.Logger.Error("Failed to send transfer file: %v", err)

		return c.wrapAndSendError(err)
	}

	return nil
}

func (c *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	eot, err := c.ses.Recv(clientWriter{w: file})
	if err != nil {
		c.ses = nil
		c.pip.Logger.Error("Failed to receive transfer file: %v", err)

		return c.wrapAndSendError(err)
	}

	if c.noFinalHash {
		return nil
	}

	hash, hErr := internal.MakeHash(c.ctx, c.pip.TransCtx.FS, c.pip.Logger,
		&c.pip.TransCtx.Transfer.LocalPath)
	if hErr != nil {
		return c.wrapAndSendError(hErr)
	}

	if !bytes.Equal(eot.Hash, hash) {
		c.pip.Logger.Error("File hash does not match expected value")

		eqErr := pipeline.NewError(types.TeIntegrity, "invalid file hash")

		return c.wrapAndSendError(eqErr)
	}

	return nil
}

// EndTransfer send a transfer end message, and then closes the session.
func (c *transferClient) EndTransfer() *pipeline.Error {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address.String())

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	c.pip.Logger.Debug("Ending transfert with remote partner")

	if err := c.ses.EndRequest(); err != nil {
		c.pip.Logger.Error("Failed to end transfer request: %v", err)

		return c.wrapAndSendError(err)
	}

	return nil
}

// SendError sends the given error to the remote partner and then closes the
// session.
func (c *transferClient) SendError(code types.TransferErrorCode, msg string) {
	pErr := pipeline.NewError(code, msg)
	r66Err := internal.ToR66Error(pErr)

	c.pip.Logger.Debug("Sending error '%v' to remote partner", pErr)

	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address.String())

	if c.ses == nil {
		return
	}

	defer c.ses.Close()

	if sErr := c.ses.SendError(r66Err); sErr != nil {
		c.pip.Logger.Error("Failed to send error to remote partner: %v", sErr)
	}
}

// Pause sends a pause message to the remote partner and then closes the
// session.
func (c *transferClient) Pause() *pipeline.Error {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address.String())

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Stop(); err != nil {
		c.pip.Logger.Warning("Failed send pause signal to remote host: %v", err)

		return c.wrapAndSendError(err)
	}

	return nil
}

// Cancel sends a cancel message to the remote partner and then closes the
// session.
func (c *transferClient) Cancel() *pipeline.Error {
	defer c.cancel()
	defer c.conns.Done(c.pip.TransCtx.RemoteAgent.Address.String())

	if c.ses == nil {
		return nil
	}

	defer c.ses.Close()

	if err := c.ses.Cancel(); err != nil {
		c.pip.Logger.Warning("Failed send cancel signal to remote host: %v", err)

		return c.wrapAndSendError(err)
	}

	return nil
}

func (c *transferClient) wrapAndSendError(err error) *pipeline.Error {
	var tErr *pipeline.Error
	if !errors.As(err, &tErr) {
		tErr = internal.FromR66Error(err, c.pip)
	}

	c.SendError(tErr.Code(), tErr.Details())

	return tErr
}

type clientReader struct{ r io.ReaderAt }

func (c clientReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.r.ReadAt(p, off)
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	} else if err != nil {
		return n, internal.ToR66Error(err)
	}

	return n, nil
}

type clientWriter struct{ w io.WriterAt }

func (c clientWriter) WriteAt(p []byte, off int64) (int, error) {
	n, err := c.w.WriteAt(p, off)
	if err != nil {
		return n, internal.ToR66Error(err)
	}

	return n, nil
}
