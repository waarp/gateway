package sftp

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// transferClient is the SFTP implementation of the `pipeline.TransferClient`
// interface which enables the gateway to execute SFTP transfers.
type transferClient struct {
	pip   *pipeline.Pipeline
	conns *sftpConnPool

	sftpSession *ClientConn
	sftpFile    *sftp.File
}

func newTransferClient(pip *pipeline.Pipeline, conns *sftpConnPool) *transferClient {
	return &transferClient{pip: pip, conns: conns}
}

func (c *transferClient) Request() *pipeline.Error {
	if tErr := c.request(); tErr != nil {
		c.SendError(tErr.Code(), tErr.Details())

		return tErr
	}

	return nil
}

func (c *transferClient) request() *pipeline.Error {
	var err error
	if c.sftpSession, err = c.conns.Connect(c.pip); err != nil {
		return fromSFTPErr(err, types.TeConnection, c.pip)
	}

	filepath := c.pip.TransCtx.Transfer.RemotePath
	if c.pip.TransCtx.Rule.IsSend {
		return c.requestSend(filepath)
	}

	return c.requestReceive(filepath)
}

func (c *transferClient) requestSend(filepath string) *pipeline.Error {
	// Check parent dir, if it doesn't exist, try to create it
	parentDir := path.Dir(c.pip.TransCtx.Transfer.RemotePath)
	if stat, statErr := c.sftpSession.Stat(parentDir); statErr == nil {
		if perm := stat.Mode().Perm(); perm&0o200 == 0 {
			c.pip.Logger.Errorf("Remote parent directory %q is not writable (permissions %s)",
				parentDir, perm.String())

			return pipeline.NewError(types.TeForbidden, "cannot write to remote parent directory")
		}
	} else if !os.IsNotExist(statErr) {
		c.pip.Logger.Warningf("Failed to check parent directory: %v", statErr)
	}

	if mkdirErr := c.sftpSession.MkdirAll(parentDir); mkdirErr != nil {
		c.pip.Logger.Warningf("Failed to create remote parent directory: %v", mkdirErr)
	}

	if c.pip.TransCtx.Transfer.Progress > 0 {
		if stat, statErr := c.sftpSession.Stat(filepath); statErr != nil {
			c.pip.Logger.Warningf("Failed to retrieve the remote file's size: %v", statErr)
			c.pip.TransCtx.Transfer.Progress = 0
		} else {
			c.pip.TransCtx.Transfer.Progress = stat.Size()
		}

		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	return nil
}

func (c *transferClient) requestReceive(filepath string) *pipeline.Error {
	if stat, statErr := c.sftpSession.Stat(filepath); os.IsNotExist(statErr) {
		c.pip.Logger.Errorf("Remote file %q does not exist", filepath)

		return fromSFTPErr(statErr, types.TeUnknownRemote, c.pip)
	} else if statErr != nil {
		c.pip.Logger.Warningf("Failed to check the remote file %q: %v", filepath, statErr)
	} else if perm := stat.Mode().Perm(); perm&0o400 == 0 {
		c.pip.Logger.Errorf("Remote file %q is not readable (permissions %s)",
			filepath, perm.String())

		return pipeline.NewError(types.TeForbidden, "remote file is not readable")
	}

	return nil
}

// Send copies the content from the local source file to the remote one.
func (c *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	filepath := c.pip.TransCtx.Transfer.RemotePath

	var openErr error
	if c.sftpFile, openErr = c.sftpSession.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC); openErr != nil {
		c.pip.Logger.Errorf("Failed to create remote file: %v", openErr)

		return fromSFTPErr(openErr, types.TeUnknownRemote, c.pip)
	}

	if _, err := c.sftpFile.ReadFrom(file); err != nil {
		c.pip.Logger.Errorf("Failed to write to remote SFTP file: %v", err)

		return c.wrapAndSendError(err, types.TeDataTransfer)
	}

	if err := c.sftpFile.Close(); err != nil {
		c.pip.Logger.Errorf("Failed to close remote SFTP file: %v", err)
		return fromSFTPErr(err, types.TeFinalization, c.pip)
	}

	return nil
}

func (c *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	filepath := c.pip.TransCtx.Transfer.RemotePath

	var openErr error
	if c.sftpFile, openErr = c.sftpSession.Open(filepath); openErr != nil {
		c.pip.Logger.Errorf("Failed to open remote file: %v", openErr)

		return fromSFTPErr(openErr, types.TeUnknownRemote, c.pip)
	}

	if c.pip.TransCtx.Transfer.Progress != 0 {
		if _, err := c.sftpFile.Seek(c.pip.TransCtx.Transfer.Progress, io.SeekStart); err != nil {
			c.pip.Logger.Errorf("Failed to seek into remote SFTP file: %v", err)

			return c.wrapAndSendError(err, types.TeUnknownRemote)
		}
	}

	if _, err := c.sftpFile.WriteTo(file); err != nil {
		c.pip.Logger.Errorf("Failed to read from remote SFTP file: %v", err)

		return c.wrapAndSendError(err, types.TeDataTransfer)
	}

	if err := c.sftpFile.Close(); err != nil {
		c.pip.Logger.Errorf("Failed to close remote SFTP file: %v", err)

		return fromSFTPErr(err, types.TeFinalization, c.pip)
	}

	return nil
}

func (c *transferClient) EndTransfer() *pipeline.Error {
	return c.endTransfer()
}

func (c *transferClient) endTransfer() (tErr *pipeline.Error) {
	c.conns.CloseConn(c.pip)

	return nil
}

func (c *transferClient) wrapAndSendError(err error, defaultCode types.TransferErrorCode) *pipeline.Error {
	tErr := fromSFTPErr(err, defaultCode, c.pip)
	c.SendError(tErr.Code(), tErr.Details())

	return tErr
}

func (c *transferClient) SendError(types.TransferErrorCode, string) {
	//nolint:errcheck //error is irrelevant here
	_ = c.endTransfer()
}

func (c *transferClient) Delete(ctx context.Context, filepath string) error {
	return utils.RunWithCtx(ctx, func() error {
		return c.sftpSession.Remove(filepath)
	})
}

func (c *transferClient) DeleteAll(ctx context.Context, filepath string) error {
	return utils.RunWithCtx(ctx, func() error {
		return c.sftpSession.RemoveAll(filepath)
	})
}
