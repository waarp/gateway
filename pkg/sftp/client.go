// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func init() {
	executor.ClientsConstructors["sftp"] = NewClient
}

// Client is the SFTP implementation of the `pipeline.Client` interface which
// enables the gateway to initiate SFTP transfers.
type Client struct {
	Signals <-chan model.Signal
	Info    model.OutTransferInfo

	conf       *config.SftpProtoConfig
	conn       net.Conn
	client     *sftp.Client
	remoteFile *sftp.File
}

// NewClient returns a new SFTP transfer client with the given transfer info,
// local file, and signal channel. An error is returned if the client
// configuration is incorrect.
func NewClient(info model.OutTransferInfo, signals <-chan model.Signal) (pipeline.Client, error) {

	client := &Client{
		Info:    info,
		Signals: signals,
	}

	conf := &config.SftpProtoConfig{}
	if err := json.Unmarshal(info.Agent.ProtoConfig, conf); err != nil {
		return nil, err
	}
	client.conf = conf

	return client, nil
}

// Connect opens a TCP connection to the remote.
func (c *Client) Connect() *model.PipelineError {
	conn, err := net.Dial("tcp", c.Info.Agent.Address)
	if err != nil {
		return model.NewPipelineError(types.TeConnection, err.Error())
	}
	c.conn = conn

	return nil
}

// Authenticate opens the SSH tunnel to the remote.
func (c *Client) Authenticate() *model.PipelineError {
	conf, err := getSSHClientConfig(&c.Info, c.conf)
	if err != nil {
		return model.NewPipelineError(types.TeInternal, err.Error())
	}

	addr, _, _ := net.SplitHostPort(c.Info.Agent.Address)
	conn, chans, reqs, err := ssh.NewClientConn(c.conn, addr, conf)
	if err != nil {
		return model.NewPipelineError(types.TeBadAuthentication, err.Error())
	}

	sshClient := ssh.NewClient(conn, chans, reqs)
	c.client, err = sftp.NewClient(sshClient)
	if err != nil {
		return model.NewPipelineError(types.TeConnection, err.Error())
	}
	return nil
}

// Request opens/creates the remote file.
func (c *Client) Request() *model.PipelineError {
	var err error
	if c.Info.Rule.IsSend {
		remotePath := path.Join(c.Info.Rule.InPath, c.Info.Transfer.DestFile)
		c.remoteFile, err = c.client.Create(remotePath)
		if err != nil {
			if msg, ok := isRemoteTaskError(err); ok {
				fullMsg := fmt.Sprintf("Remote pre-tasks failed: %s", msg)
				return model.NewPipelineError(types.TeExternalOperation, fullMsg)
			}
			if err == os.ErrNotExist {
				return model.NewPipelineError(types.TeFileNotFound, "Target directory does not exist")
			}
			return model.NewPipelineError(types.TeConnection, err.Error())
		}
	} else {
		remotePath := path.Join(c.Info.Rule.OutPath, c.Info.Transfer.SourceFile)
		c.remoteFile, err = c.client.Open(remotePath)
		if err != nil {
			if msg, ok := isRemoteTaskError(err); ok {
				fullMsg := fmt.Sprintf("Remote pre-tasks failed: %s", msg)
				return model.NewPipelineError(types.TeExternalOperation, fullMsg)
			}
			if err == os.ErrNotExist {
				return model.NewPipelineError(types.TeFileNotFound, "Target file does not exist")
			}
			return model.NewPipelineError(types.TeConnection, err.Error())
		}
	}
	return nil
}

// Data copies the content of the source file into the destination file.
func (c *Client) Data(file pipeline.DataStream) *model.PipelineError {
	err := func() error {
		if !c.Info.Rule.IsSend {
			_, err := c.remoteFile.WriteTo(file)
			return err
		}
		if _, err := c.remoteFile.ReadFrom(file); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return model.NewPipelineError(types.TeDataTransfer, err.Error())
	}
	return nil
}

// Close ends the SFTP session and closes the connection.
func (c *Client) Close(pErr *model.PipelineError) *model.PipelineError {
	defer func() {
		if c.client != nil {
			_ = c.client.Close()
		}
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}()

	if pErr == nil {
		if err := c.remoteFile.Close(); err != nil {
			if msg, ok := isRemoteTaskError(err); ok {
				fullMsg := fmt.Sprintf("Remote post-tasks failed: %s", msg)
				return model.NewPipelineError(types.TeExternalOperation, fullMsg)
			}
			return model.NewPipelineError(types.TeConnection, err.Error())
		}
	}
	return nil
}
