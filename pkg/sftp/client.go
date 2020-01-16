// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

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
	Signals <-chan pipeline.Signal
	Info    model.OutTransferInfo

	conf       *config.SftpProtoConfig
	conn       net.Conn
	client     *sftp.Client
	localFile  *os.File
	remoteFile *sftp.File
}

// NewClient returns a new SFTP transfer client with the given transfer info,
// local file, and signal channel. An error is returned if the client
// configuration is incorrect.
func NewClient(info model.OutTransferInfo, localFile *os.File,
	signals <-chan pipeline.Signal) (pipeline.Client, error) {

	client := &Client{
		Info:      info,
		Signals:   signals,
		localFile: localFile,
	}

	conf := &config.SftpProtoConfig{}
	if err := json.Unmarshal(info.Agent.ProtoConfig, conf); err != nil {
		return nil, err
	}
	client.conf = conf

	return client, nil
}

// Connect opens a TCP connection to the remote.
func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.conf.Address, c.conf.Port))
	if err != nil {
		return err
	}
	c.conn = conn

	return nil
}

// Authenticate opens the SSH tunnel to the remote.
func (c *Client) Authenticate() error {
	if c.conn == nil {
		return fmt.Errorf("no connection to authenticate")
	}

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
	return fmt.Errorf("no valid certificate found")
}

// Request opens/creates the remote file.
func (c *Client) Request() (err error) {
	if c.Info.Rule.IsSend {
		c.remoteFile, err = c.client.Create(c.Info.Transfer.DestPath)
	} else {
		c.remoteFile, err = c.client.Open(c.Info.Transfer.SourcePath)
	}
	return
}

// Data copies the content of the source file into the destination file.
func (c *Client) Data() error {
	defer func() {
		_ = c.localFile.Close()
		_ = c.remoteFile.Close()
		_ = c.client.Close()
		_ = c.conn.Close()
	}()

	if c.Info.Rule.IsSend {
		if _, err := c.remoteFile.ReadFrom(c.localFile); err != nil {
			return err
		}
	} else {
		if _, err := c.remoteFile.WriteTo(c.localFile); err != nil {
			return err
		}
	}
	return nil
}
