package r66

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ClientDialTimeout = 10 * time.Second

type r66ConnPool = protoutils.ConnPool[*clientConn]

type clientConn struct {
	*r66.Client
}

func (c *clientConn) Close() error {
	c.Client.Close()

	return nil
}

func makeDialer(client *model.Client) (*protoutils.TraceDialer, error) {
	dialer := &net.Dialer{Timeout: ClientDialTimeout}

	if client.LocalAddress.IsSet() {
		var err error
		if dialer.LocalAddr, err = net.ResolveTCPAddr("tcp",
			client.LocalAddress.String()); err != nil {
			return nil, fmt.Errorf("failed to parse the R66 client's local address: %w", err)
		}
	}

	return &protoutils.TraceDialer{Dialer: dialer}, nil
}

func (c *Client) dialClientConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*clientConn, error) {
	var partConf tlsPartnerConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partConf); err != nil {
		pip.Logger.Errorf("Failed to parse R66 partner proto config: %v", err)

		return nil, pipeline.NewErrorWith(types.TeInternal,
			"failed to parse R66 partner proto config", err)
	}

	var tlsConf *tls.Config
	if c.cli.Protocol == R66TLS {
		var err error
		if tlsConf, err = makeClientTLSConfig(pip, &partConf, c.clientConfig); err != nil {
			c.logger.Errorf("Failed to parse R66 TLS config: %v", err)

			return nil, pipeline.NewErrorWith(types.TeInternal, "invalid R66 TLS config", err)
		}
	}

	addr := conf.GetRealAddress(pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(pip.TransCtx.RemoteAgent.Address.Port))

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate the TCP connection: %w", err)
	}

	if tlsConf != nil {
		conn = tls.Client(conn, tlsConf)
	}

	client, err := r66.NewClient(conn, c.logger.AsStdLogger(log.LevelTrace))
	if err != nil {
		return nil, fmt.Errorf("failed to initiate the R66 connection: %w", err)
	}

	return &clientConn{client}, nil
}
