// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"context"
	"fmt"
	"net"
	"time"

	"code.waarp.fr/lib/log"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const clientDialTimeout = 10 * time.Second

// client is the SFTP implementation of the `pipeline.client` interface which
// enables the gateway to initiate SFTP transfers.
type client struct {
	db     *database.DB
	client *model.Client

	state   utils.State
	logger  *log.Logger
	sshConf ssh.Config
	dialer  *net.Dialer
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.client.Name)
	if err := c.start(); err != nil {
		c.logger.Error("Failed to start the client: %v", err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) start() error {
	var clientConf clientConfig
	if err := utils.JSONConvert(c.client.ProtoConfig, &clientConf); err != nil {
		return fmt.Errorf("failed to parse the SFTP client's proto config: %w", err)
	}

	c.dialer = &net.Dialer{Timeout: clientDialTimeout}
	c.sshConf = ssh.Config{
		KeyExchanges: clientConf.KeyExchanges,
		Ciphers:      clientConf.Ciphers,
		MACs:         clientConf.MACs,
	}

	if c.client.LocalAddress != "" {
		var err error

		c.dialer.LocalAddr, err = net.ResolveTCPAddr("tcp", c.client.LocalAddress)
		if err != nil {
			return fmt.Errorf("failed to parse the SFTP client's local address: %w", err)
		}
	}

	return nil
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return newTransferClient(pip, c.dialer, &c.sshConf)
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, c.client.ID); err != nil {
		c.logger.Error("Failed to stop the SFTP client: %v", err)
		c.state.Set(utils.StateError, err.Error())

		return fmt.Errorf("failed to stop the SFTP client: %w", err)
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}
