package sftp

import (
	"io"
	"os"
	"path"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

// transferClient is the SFTP implementation of the `pipeline.TransferClient`
// interface which enables the gateway to execute SFTP transfers.
type transferClient struct {
	pip   *pipeline.Pipeline
	conns *sftpConnPool

	sftpSession *clientConn
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

	var err error

	c.sftpFile, err = c.sftpSession.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		c.pip.Logger.Errorf("Failed to create remote file: %v", err)

		return fromSFTPErr(err, types.TeUnknownRemote, c.pip)
	}

	return nil
}

func (c *transferClient) requestReceive(filepath string) *pipeline.Error {
	var err error

	c.sftpFile, err = c.sftpSession.Open(filepath)
	if err != nil {
		c.pip.Logger.Errorf("Failed to open remote file: %v", err)

		return fromSFTPErr(err, types.TeUnknownRemote, c.pip)
	}

	return nil
}

// Send copies the content from the local source file to the remote one.
func (c *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	// Check parent dir, if it doesn't exist, try to create it
	parentDir := path.Dir(c.pip.TransCtx.Transfer.RemotePath)
	if mkdirErr := c.sftpSession.MkdirAll(parentDir); mkdirErr != nil {
		c.pip.Logger.Errorf("Failed to create remote parent directory: %v", mkdirErr)

		return c.wrapAndSendError(mkdirErr, types.TeUnknownRemote)
	}

	if _, err := c.sftpFile.ReadFrom(file); err != nil {
		c.pip.Logger.Errorf("Failed to write to remote SFTP file: %v", err)

		return c.wrapAndSendError(err, types.TeDataTransfer)
	}

	return nil
}

func (c *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
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

	return nil
}

func (c *transferClient) EndTransfer() *pipeline.Error {
	return c.endTransfer()
}

func (c *transferClient) endTransfer() (tErr *pipeline.Error) {
	defer c.conns.CloseConn(c.pip)

	if c.sftpFile != nil {
		if err := c.sftpFile.Close(); err != nil {
			c.pip.Logger.Errorf("Failed to close remote SFTP file: %v", err)

			if cErr := c.sftpSession.Close(); cErr != nil {
				c.pip.Logger.Warningf("An error occurred while closing the SFTP session: %v", cErr)
			}

			tErr = fromSFTPErr(err, types.TeFinalization, c.pip)
		}
	}

	if c.sftpSession != nil {
		if err := c.sftpSession.Close(); err != nil {
			c.pip.Logger.Errorf("Failed to close SFTP session: %v", err)

			if tErr == nil {
				tErr = fromSFTPErr(err, types.TeFinalization, c.pip)
			}
		}
	}

	return tErr
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
