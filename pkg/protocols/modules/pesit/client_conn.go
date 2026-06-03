package pesit

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/lib/pesit"
)

const pesitDialTimeout = 10 * time.Second

type pesitConnPool = protoutils.ConnPool[*pesitClientConn]

// pesitClientConn wraps a pesit.Client to satisfy the io.Closer interface

// required by ConnPool. On Close(), it sends F.RELEASE to properly

// terminate the PeSIT connection.

type pesitClientConn struct {
	*pesit.Client
}

func (c *pesitClientConn) Close() error {
	if err := c.Client.Close(nil); err != nil {
		return fmt.Errorf("failed to close PeSIT connection: %w", err)
	}

	return nil
}

// dialPesitConn is the ConnPool openConn function. It dials a new TCP

// connection (optionally with TLS), creates a pesit.Client, configures

// it, and performs the F.CONNECT handshake + authentication.

func (c *client) dialPesitConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*pesitClientConn, error) {
	var partConf PartnerConfigTLS

	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partConf); err != nil {

		pip.Logger.Errorf("Failed to parse PeSIT partner proto config: %v", err)

		return nil, pipeline.NewErrorWith(types.TeInternal, "failed to parse PeSIT partner proto config", err)

	}

	// Dial TCP connection

	addr := conf.GetRealAddress(pip.TransCtx.RemoteAgent.Address.Host,

		utils.FormatUint(pip.TransCtx.RemoteAgent.Address.Port))

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to partner: %w", err)
	}

	// Optional TLS handshake

	if c.isTLS {

		ct := &clientTransfer{pip: pip}

		tlsConfig, tlsErr := ct.makeTLSConfig(pip.TransCtx.RemoteAgent.Address.Host, &partConf)

		if tlsErr != nil {

			conn.Close()

			return nil, fmt.Errorf("failed to create TLS config: %w", tlsErr)

		}

		conn = tls.Client(conn, tlsConfig)

	}

	// Create PeSIT client

	serverLogin := pip.TransCtx.RemoteAgent.Name

	if partConf.Login != "" {
		serverLogin = partConf.Login
	}

	pesitClient := pesit.NewClient(pip.TransCtx.RemoteAccount.Login,

		getPassword(pip.TransCtx), serverLogin)

	pesitClient.Logger = pip.Logger.AsStdLogger(log.LevelDebug)

	pesitClient.NetworkTrace = pip.Logger.AsStdLogger(log.LevelTrace)

	// Configure client options

	configurePesitClient(pesitClient, &partConf.PartnerConfig, c.conf, pip, c.isTLS)

	// Perform F.CONNECT handshake

	if connErr := pesitClient.Connect(conn); connErr != nil {

		conn.Close()

		return nil, fmt.Errorf("failed to establish PeSIT connection: %w", connErr)

	}

	// Authenticate server

	if authErr := authenticateServerConn(pesitClient, pip); authErr != nil {

		pesitClient.Close(nil)

		return nil, fmt.Errorf("server authentication failed: %w", authErr)

	}

	pip.Logger.Debug("PeSIT connection established and pooled")

	return &pesitClientConn{pesitClient}, nil
}

// configurePesitClient applies the partner and client config to a pesit.Client.

func configurePesitClient(
	client *pesit.Client, config *PartnerConfig,

	clientConf *ClientConfigTLS, pip *pipeline.Pipeline, isTLS bool,
) {
	if utils.If(config.DisableCheckpoints.Valid,

		config.DisableCheckpoints.Value, clientConf.DisableCheckpoints) {

		client.AllowCheckpoints(pesit.CheckpointDisabled, 0)
	} else {

		client.AllowCheckpoints(

			utils.If(config.CheckpointSize != 0,

				config.CheckpointSize,

				clientConf.CheckpointSize),

			utils.If(config.CheckpointWindow != 0,

				config.CheckpointWindow,

				clientConf.CheckpointWindow),
		)

		client.AllowRestart(!utils.If(config.DisableRestart.Valid,

			config.DisableRestart.Value, clientConf.DisableRestart))

	}

	if config.UseNSDU.Valid && config.UseNSDU.Value {
		client.SetNSDUUsage(true)
	}

	if config.CompatibilityMode == CompatibilityModeHistorique {
		client.SetHistoriqueMode(true)
	}

	usePreConn := !config.DisablePreConnection && !isTLS

	for _, cred := range pip.TransCtx.RemoteAccountCreds {

		if cred.Type != PreConnectionAuth {
			continue
		}

		usePreConn = true

		client.SetPreConnectLogin(cred.Value)

		client.SetPreConnectPassword(cred.Value2)

		break

	}

	client.SetPreConnectionUsage(usePreConn)

	if config.ProtocolTimeout > 0 {
		client.SetMonitoringTimeout(config.ProtocolTimeout)
	}

	//nolint:errcheck // freetext is best-effort

	setFreetext(pip, clientConnFreetextKey, client)
}

var (
	errServerPasswordChange = errors.New("server password change not allowed")

	errServerAuthFailed = errors.New("server authentication failed: bad password")
)

// authenticateServerConn checks the server password returned in ACONNECT.

func authenticateServerConn(client *pesit.Client, pip *pipeline.Pipeline) error {
	if client.NewServerPassword() != "" {
		return errServerPasswordChange
	}

	var servPwd model.Credential

	if err := pip.DB.Get(&servPwd, "remote_agent_id=? AND type=?",

		pip.TransCtx.RemoteAgent.ID, auth.Password).Run(); err == nil {
		if bcrypt.CompareHashAndPassword([]byte(servPwd.Value), []byte(client.ServerPassword())) != nil {
			return errServerAuthFailed
		}
	}

	return nil
}

func makePesitDialer(cli *model.Client) *protoutils.TraceDialer {
	dialer := &net.Dialer{Timeout: pesitDialTimeout}

	if cli.LocalAddress.IsSet() {

		addr, err := net.ResolveTCPAddr("tcp", cli.LocalAddress.String())

		if err == nil {
			dialer.LocalAddr = addr
		}

	}

	return &protoutils.TraceDialer{Dialer: dialer}
}
