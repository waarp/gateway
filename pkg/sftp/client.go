// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/executor"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

//nolint:gochecknoinits // designed to use inits
func init() {
	executor.ClientsConstructors["sftp"] = NewClient
}

// Client is the SFTP implementation of the `pipeline.Client` interface which
// enables the gateway to initiate SFTP transfers.
type Client struct {
	Signals <-chan model.Signal
	Info    *model.OutTransferInfo

	conf       *config.SftpProtoConfig
	conn       net.Conn
	client     *sftp.Client
	remoteFile *sftp.File
}

// NewClient returns a new SFTP transfer client with the given transfer info,
// local file, and signal channel. An error is returned if the client
// configuration is incorrect.
func NewClient(info *model.OutTransferInfo, signals <-chan model.Signal) (pipeline.Client, error) {
	client := &Client{
		Info:    info,
		Signals: signals,
	}

	conf := &config.SftpProtoConfig{}
	if err := json.Unmarshal(info.Agent.ProtoConfig, conf); err != nil {
		return nil, fmt.Errorf("cannot parse protoconfig for %q: %w", info.Agent.Name, err)
	}

	client.conf = conf

	return client, nil
}

// Connect opens a TCP connection to the remote.
func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.Info.Agent.Address)
	if err != nil {
		return types.NewTransferError(types.TeConnection, err.Error())
	}

	c.conn = conn

	return nil
}

// Authenticate opens the SSH tunnel to the remote.
func (c *Client) Authenticate() error {
	conf, err := getSSHClientConfig(c.Info, c.conf)
	if err != nil {
		return types.NewTransferError(types.TeInternal, err.Error())
	}

	addr, _, err := net.SplitHostPort(c.Info.Agent.Address)
	if err != nil {
		return types.NewTransferError(types.TeInternal, err.Error())
	}

	conn, chans, reqs, err := ssh.NewClientConn(c.conn, addr, conf)
	if err != nil {
		return types.NewTransferError(types.TeBadAuthentication, err.Error())
	}

	sshClient := ssh.NewClient(conn, chans, reqs)

	c.client, err = sftp.NewClient(sshClient)
	if err != nil {
		return types.NewTransferError(types.TeConnection, err.Error())
	}

	return nil
}

// Request opens/creates the remote file.
func (c *Client) Request() error {
	var err error

	if c.Info.Rule.IsSend {
		err = doSendRequest(c)
	} else {
		err = doRecvRequest(c)
	}

	return err
}

func doSendRequest(c *Client) error {
	remotePath := path.Join(c.Info.Rule.InPath, c.Info.Transfer.DestFile)

	remoteFile, err := c.client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		if msg, ok := isRemoteTaskError(err); ok {
			fullMsg := fmt.Sprintf("Remote pre-tasks failed: %s", msg)

			return types.NewTransferError(types.TeExternalOperation, fullMsg)
		}

		if errors.Is(err, os.ErrNotExist) {
			return types.NewTransferError(types.TeFileNotFound, "Target directory does not exist")
		}

		return types.NewTransferError(types.TeConnection, err.Error())
	}

	c.remoteFile = remoteFile

	return nil
}

func doRecvRequest(c *Client) error {
	remotePath := path.Join(c.Info.Rule.OutPath, c.Info.Transfer.SourceFile)

	remoteFile, err := c.client.Open(remotePath)
	if err != nil {
		if msg, ok := isRemoteTaskError(err); ok {
			fullMsg := fmt.Sprintf("Remote pre-tasks failed: %s", msg)

			return types.NewTransferError(types.TeExternalOperation, fullMsg)
		}

		if errors.Is(err, os.ErrNotExist) {
			return types.NewTransferError(types.TeFileNotFound, "Target file does not exist")
		}

		return types.NewTransferError(types.TeConnection, err.Error())
	}

	c.remoteFile = remoteFile

	return nil
}

// Data copies the content of the source file into the destination file.
func (c *Client) Data(file pipeline.DataStream) error {
	if !c.Info.Rule.IsSend {
		_, err2 := c.remoteFile.WriteTo(file)
		if err2 != nil {
			return types.NewTransferError(types.TeDataTransfer,
				fmt.Sprintf("cannot write data: %v", err2))
		}

		return nil
	}

	_, err := c.remoteFile.ReadFrom(file)
	if err != nil {
		return types.NewTransferError(types.TeDataTransfer,
			fmt.Sprintf("cannot read data: %v", err))
	}

	return nil
}

// Close ends the SFTP session and closes the connection.
func (c *Client) Close(pErr error) error {
	defer func() {
		if c.client != nil {
			_ = c.client.Close() //nolint:errcheck // no logger to do anything with the error
		}

		if c.conn != nil {
			_ = c.conn.Close() //nolint:errcheck // no logger to do anything with the error
		}
	}()

	if pErr == nil && c.remoteFile != nil {
		if err := c.remoteFile.Close(); err != nil {
			if msg, ok := isRemoteTaskError(err); ok {
				fullMsg := fmt.Sprintf("Remote post-tasks failed: %s", msg)

				return types.NewTransferError(types.TeExternalOperation, fullMsg)
			}

			return types.NewTransferError(types.TeConnection, err.Error())
		}
	}

	return nil
}
