// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func init() {
	pipeline.ClientConstructors["sftp"] = NewClient
}

// client is the SFTP implementation of the `pipeline.Client` interface which
// enables the gateway to initiate SFTP transfers.
type client struct {
	logger   *log.Logger
	transCtx *model.TransferContext

	sshConf *ssh.ClientConfig

	sshSession  *ssh.Client
	sftpSession *sftp.Client
	remoteFile  *sftp.File
}

// NewClient returns a new SFTP transfer client with the given transfer info,
// local file, and signal channel. An error is returned if the client
// configuration is incorrect.
func NewClient(logger *log.Logger, transCtx *model.TransferContext) (pipeline.Client, error) {
	conf := &config.SftpProtoConfig{}
	if err := json.Unmarshal(transCtx.RemoteAgent.ProtoConfig, conf); err != nil {
		logger.Errorf("Failed to parse SFTP partner protocol configuration: %s", err)
		return nil, types.NewTransferError(types.TeInternal,
			"failed to parse SFTP partner protocol configuration")
	}

	sshConf, err := getSSHClientConfig(transCtx, conf)
	if err != nil {
		logger.Errorf("Failed to make SFTP client configuration: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to make SFTP configuration")
	}

	return &client{
		logger:   logger,
		transCtx: transCtx,
		sshConf:  sshConf,
	}, nil
}

func (c *client) Request() error {
	var err error
	c.sshSession, err = ssh.Dial("tcp", c.transCtx.RemoteAgent.Address, c.sshConf)
	if err != nil {
		c.logger.Errorf("Failed to connect to SFTP host: %s", err)
		return types.NewTransferError(sftpToCode(err, types.TeConnection),
			"failed to connect to SFTP host")
	}

	c.sftpSession, err = sftp.NewClient(c.sshSession)
	if err != nil {
		c.logger.Errorf("Failed to start SFTP session: %s", err)
		return types.NewTransferError(sftpToCode(err, types.TeUnknownRemote),
			"failed to start SFTP session")
	}

	if c.transCtx.Rule.IsSend {
		c.remoteFile, err = c.sftpSession.Create(c.transCtx.Transfer.RemotePath)
		if err != nil {
			c.logger.Errorf("Failed to create remote file: %s", err)
			return types.NewTransferError(sftpToCode(err, types.TeUnknownRemote),
				"failed to create remote file")
		}
	} else {
		c.remoteFile, err = c.sftpSession.Open(c.transCtx.Transfer.RemotePath)
		if err != nil {
			c.logger.Errorf("Failed to open remote file: %s", err)
			return types.NewTransferError(sftpToCode(err, types.TeUnknownRemote),
				"failed to open remote file")
		}
	}
	return nil
}

// Data copies the content of the source file into the destination file.
func (c *client) Data(data pipeline.TransferStream) error {
	if c.transCtx.Transfer.Progress != 0 {
		_, err := c.remoteFile.Seek(int64(c.transCtx.Transfer.Progress), io.SeekStart)
		if err != nil {
			c.logger.Errorf("Failed to seek into remote SFTP file: %s", err)
			return types.NewTransferError(sftpToCode(err, types.TeUnknownRemote),
				"failed to seek info SFTP file")
		}
	}

	if c.transCtx.Rule.IsSend {
		_, err := c.remoteFile.ReadFrom(data)
		if err != nil {
			c.logger.Errorf("Failed to write to remote SFTP file: %s", err)
			return types.NewTransferError(sftpToCode(err, types.TeDataTransfer),
				"failed to write to SFTP file")
		}
	} else {
		_, err := c.remoteFile.WriteTo(data)
		if err != nil {
			c.logger.Errorf("Failed to read from remote SFTP file: %s", err)
			return types.NewTransferError(sftpToCode(err, types.TeDataTransfer),
				"failed to read from SFTP file")
		}
	}
	return nil
}

func (c *client) EndTransfer() error {
	defer func() {
		_ = c.sshSession.Close()
	}()
	if err := c.remoteFile.Close(); err != nil {
		c.logger.Errorf("Failed to close remote SFTP file: %s", err)
		return types.NewTransferError(sftpToCode(err, types.TeFinalization),
			"failed to close remote SFTP file")
	}
	if err := c.sftpSession.Close(); err != nil {
		c.logger.Errorf("Failed to close SFTP session: %s", err)
		return types.NewTransferError(sftpToCode(err, types.TeFinalization),
			"failed to close SFTP session")
	}
	return nil
}

func (c *client) SendError(error) {
	_ = c.sshSession.Close()
}
