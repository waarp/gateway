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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
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
	conns   *protoutils.ConnPool[*clientConn]
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
		c.logger.Errorf("Failed to start the client: %v", err)
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.client.Name, err)

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

	c.sshConf = ssh.Config{
		KeyExchanges: clientConf.KeyExchanges,
		Ciphers:      clientConf.Ciphers,
		MACs:         clientConf.MACs,
	}

	dialer := &protoutils.TraceDialer{Dialer: &net.Dialer{Timeout: clientDialTimeout}}
	if c.client.LocalAddress.IsSet() {
		var err error
		if dialer.LocalAddr, err = net.ResolveTCPAddr("tcp", c.client.LocalAddress.String()); err != nil {
			return fmt.Errorf("failed to parse the SFTP client's local address: %w", err)
		}
	}

	c.conns = protoutils.NewConnPool[*clientConn](dialer, c.newClientConn)

	return nil
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return newTransferClient(pip, c.conns), nil
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, c.client.ID); err != nil {
		c.logger.Errorf("Failed to stop the SFTP client: %v", err)
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.client.Name, err)

		return fmt.Errorf("failed to stop the SFTP client: %w", err)
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) SetGracePeriod(d time.Duration) {
	c.conns.SetGracePeriod(d)
}
