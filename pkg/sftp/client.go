// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

const ClientDialTimeout = 10 * time.Second

// client is the SFTP implementation of the `pipeline.Client` interface which
// enables the gateway to initiate SFTP transfers.
type client struct {
	transfers *service.TransferMap
	client    *model.Client
	sshConf   ssh.Config
	dialer    *net.Dialer
	state     state.State
}

func NewClient(dbClient *model.Client) (pipeline.Client, error) {
	var clientConf config.SftpClientProtoConfig
	if err := utils.JSONConvert(dbClient.ProtoConfig, &clientConf); err != nil {
		return nil, fmt.Errorf("failed to parse the SFTP client's proto config: %w", err)
	}

	cli := &client{
		transfers: service.NewTransferMap(),
		client:    dbClient,
		dialer:    &net.Dialer{Timeout: ClientDialTimeout},
		sshConf: ssh.Config{
			KeyExchanges: clientConf.KeyExchanges,
			Ciphers:      clientConf.Ciphers,
			MACs:         clientConf.MACs,
		},
	}

	if dbClient.LocalAddress != "" {
		var err error

		cli.dialer.LocalAddr, err = net.ResolveTCPAddr("tcp", dbClient.LocalAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the SFTP client's local address: %w", err)
		}
	}

	return cli, nil
}

func (c *client) Start() error {
	c.state.Set(state.Running, "")

	return nil
}

func (c *client) State() *state.State { return &c.state }

func (c *client) ManageTransfers() *service.TransferMap { return c.transfers }

func (c *client) InitTransfer(pip *pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	return newTransferClient(pip, c.dialer, &c.sshConf)
}

func (c *client) Stop(ctx context.Context) error {
	if err := c.transfers.InterruptAll(ctx); err != nil {
		return fmt.Errorf("failed to stop the running transfers: %w", err)
	}

	c.state.Set(state.Offline, "")

	return nil
}
