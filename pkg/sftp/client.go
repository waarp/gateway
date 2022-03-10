// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

//nolint:gochecknoinits // designed to use inits
func init() {
	pipeline.ClientConstructors["sftp"] = NewClient
}

// client is the SFTP implementation of the `pipeline.Client` interface which
// enables the gateway to initiate SFTP transfers.
type client struct {
	pip *pipeline.Pipeline

	protoConf   *config.SftpProtoConfig
	sshConf     *ssh.ClientConfig
	sshSession  *ssh.Client
	sftpSession *sftp.Client
	remoteFile  *sftp.File
}

// NewClient returns a new SFTP transfer client with the given transfer info,
// local file, and signal channel. An error is returned if the client
// configuration is incorrect.
func NewClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return newClient(pip)
}

func newClient(pip *pipeline.Pipeline) (*client, *types.TransferError) {
	var protoConf config.SftpProtoConfig
	if err := json.Unmarshal(pip.TransCtx.RemoteAgent.ProtoConfig, &protoConf); err != nil {
		pip.Logger.Errorf("Failed to parse SFTP partner protocol configuration: %s", err)

		return nil, types.NewTransferError(types.TeInternal,
			"failed to parse SFTP partner protocol configuration")
	}

	sshConf, err := getSSHClientConfig(pip.TransCtx, &protoConf)
	if err != nil {
		pip.Logger.Errorf("Failed to make SFTP client configuration: %s", err)

		return nil, types.NewTransferError(types.TeInternal, "failed to make SFTP configuration")
	}

	return &client{
		pip:       pip,
		protoConf: &protoConf,
		sshConf:   sshConf,
	}, nil
}

func (c *client) Request() (tErr *types.TransferError) {
	defer func() {
		if tErr != nil {
			if cErr := c.EndTransfer(); cErr != nil {
				c.pip.Logger.Warningf("Failed to end SFTP transfer: %v", cErr)
			}
		}
	}()

	addr := c.pip.TransCtx.RemoteAgent.Address

	var err error

	c.sshSession, err = ssh.Dial("tcp", addr, c.sshConf)
	if err != nil {
		c.pip.Logger.Errorf("Failed to connect to SFTP host: %s", err)

		return c.fromSFTPErr(err, types.TeConnection)
	}

	var opts []sftp.ClientOption

	if !c.protoConf.UseStat {
		opts = append(opts, sftp.UseFstat(true))
	}

	if c.protoConf.DisableClientConcurrentReads {
		opts = append(opts, sftp.UseConcurrentReads(false))
	}

	c.sftpSession, err = sftp.NewClient(c.sshSession, opts...)
	if err != nil {
		c.pip.Logger.Errorf("Failed to start SFTP session: %s", err)

		return c.fromSFTPErr(err, types.TeUnknownRemote)
	}

	if c.pip.TransCtx.Rule.IsSend {
		c.remoteFile, err = c.sftpSession.Create(strings.TrimPrefix(
			c.pip.TransCtx.Transfer.RemotePath, "/"))
		if err != nil {
			c.pip.Logger.Errorf("Failed to create remote file: %s", err)

			return c.fromSFTPErr(err, types.TeUnknownRemote)
		}
	} else {
		c.remoteFile, err = c.sftpSession.Open(strings.TrimPrefix(
			c.pip.TransCtx.Transfer.RemotePath, "/"))
		if err != nil {
			c.pip.Logger.Errorf("Failed to open remote file: %s", err)

			return c.fromSFTPErr(err, types.TeUnknownRemote)
		}
	}

	return nil
}

// Data copies the content of the source file into the destination file.
func (c *client) Data(data pipeline.DataStream) *types.TransferError {
	if c.pip.TransCtx.Transfer.Progress != 0 {
		_, err := c.remoteFile.Seek(int64(c.pip.TransCtx.Transfer.Progress), io.SeekStart)
		if err != nil {
			c.pip.Logger.Errorf("Failed to seek into remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeUnknownRemote)
		}
	}

	if c.pip.TransCtx.Rule.IsSend {
		_, err := c.remoteFile.ReadFrom(data)
		if err != nil {
			c.pip.Logger.Errorf("Failed to write to remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeDataTransfer)
		}
	} else {
		_, err := c.remoteFile.WriteTo(data)
		if err != nil {
			c.pip.Logger.Errorf("Failed to read from remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeDataTransfer)
		}
	}

	return nil
}

func (c *client) EndTransfer() (tErr *types.TransferError) {
	if c.remoteFile != nil {
		if err := c.remoteFile.Close(); err != nil {
			c.pip.Logger.Errorf("Failed to close remote SFTP file: %s", err)

			if cErr := c.sftpSession.Close(); cErr != nil {
				c.pip.Logger.Warningf("An error occurred while closing the SFTP session: %v", cErr)
			}

			tErr = c.fromSFTPErr(err, types.TeFinalization)
		}
	}

	if c.sftpSession != nil {
		if err := c.sftpSession.Close(); err != nil {
			c.pip.Logger.Errorf("Failed to close SFTP session: %s", err)

			if tErr == nil {
				tErr = c.fromSFTPErr(err, types.TeFinalization)
			}
		}
	}

	if c.sshSession != nil {
		if err := c.sshSession.Close(); err != nil {
			c.pip.Logger.Errorf("Failed to close SSH session: %s", err)

			if tErr == nil {
				tErr = c.fromSFTPErr(err, types.TeFinalization)
			}
		}
	}

	return tErr
}

func (c *client) SendError(*types.TransferError) {
	if c.sshSession != nil {
		if err := c.sshSession.Close(); err != nil {
			c.pip.Logger.Warningf("An error occurred while closing the SSH session: %v", err)
		}
	}
}
