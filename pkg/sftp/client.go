// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"fmt"
	"io"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
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
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.conf.Address, c.conf.Port))
	if err != nil {
		return model.NewPipelineError(model.TeConnection, err.Error())
	}
	c.conn = conn

	return nil
}

// Authenticate opens the SSH tunnel to the remote.
func (c *Client) Authenticate() *model.PipelineError {
	for _, cert := range c.Info.Certs {
		conf, err := getSSHConfig(cert, c.Info.Account)
		if err != nil {
			continue
		}

		conn, chans, reqs, err := ssh.NewClientConn(c.conn, c.conf.Address, conf)
		if err != nil {
			continue
		}

		sshClient := ssh.NewClient(conn, chans, reqs)
		c.client, err = sftp.NewClient(sshClient)
		if err != nil {
			continue
		}
		return nil
	}
	return model.NewPipelineError(model.TeBadAuthentication, "no valid credentials found")
}

// Request opens/creates the remote file.
func (c *Client) Request() *model.PipelineError {
	var err error
	if c.Info.Rule.IsSend {
		c.remoteFile, err = c.client.Create(c.Info.Rule.Path + "/" + c.Info.Transfer.DestPath)
		if err != nil {
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
	} else {
		c.remoteFile, err = c.client.Open(c.Info.Transfer.SourcePath)
		if err != nil {
			return model.NewPipelineError(model.TeFileNotFound, err.Error())
		}
	}
	return nil
}

// Data copies the content of the source file into the destination file.
func (c *Client) Data(file io.ReadWriteCloser) *model.PipelineError {
	defer func() {
		_ = file.Close()
	}()

	var err error
	if c.Info.Rule.IsSend {
		_, err = c.remoteFile.ReadFrom(file)
	} else {
		_, err = c.remoteFile.WriteTo(file)
	}
	if err != nil {
		return model.NewPipelineError(model.TeDataTransfer, err.Error())
	}
	return nil
}

// Close ends the SFTP session and closes the connection.
func (c *Client) Close() *model.PipelineError {
	defer func() {
		_ = c.client.Close()
		_ = c.conn.Close()
	}()

	if err := c.remoteFile.Close(); err != nil {
		return model.NewPipelineError(model.TeExternalOperation, "Remote post-tasks failed")
	}

	return nil
}
