package ftp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"code.waarp.fr/lib/goftp"
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const clientDefaultConnTimeout = 5 * time.Second // 5s

type client struct {
	dbClient *model.Client
	state    utils.State
	logger   *log.Logger

	conf *ClientConfigTLS
}

func newClient(dbClient *model.Client) *client {
	c := &client{dbClient: dbClient}

	return c
}

func (c *client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	if err := c.start(); err != nil {
		c.state.Set(utils.StateError, err.Error())
		c.logger.Errorf("failed to start the SFTP client: %v", err)
		snmp.ReportServiceFailure(c.dbClient.Name, err)

		return err
	}

	c.state.Set(utils.StateRunning, "")

	return nil
}

func (c *client) start() error {
	c.conf = &ClientConfigTLS{}
	if err := utils.JSONConvert(c.dbClient.ProtoConfig, c.conf); err != nil {
		return fmt.Errorf("invalid client config: %w", err)
	}

	return nil
}

func (c *client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := c.stop(ctx); err != nil {
		c.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(c.dbClient.Name, err)

		return err
	}

	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *client) stop(ctx context.Context) error {
	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		c.logger.Errorf("Failed to stop the FTP client: %v", err)

		return fmt.Errorf("failed to stop the FTP client: %w", err)
	}

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	ftpClient, err := c.connect(pip)
	if err != nil {
		return nil, err
	}

	if pip.TransCtx.Rule.IsSend {
		return &clientStorTransfer{client: ftpClient, pip: pip}, nil
	}

	return &clientRetrTransfer{client: ftpClient, pip: pip}, nil
}

//nolint:funlen //too much hassle to split
func (c *client) connect(pip *pipeline.Pipeline) (*goftp.Client, *pipeline.Error) {
	partner := pip.TransCtx.RemoteAgent
	account := pip.TransCtx.RemoteAccount

	var partConf PartnerConfigTLS
	if err := utils.JSONConvert(partner.ProtoConfig, &partConf); err != nil {
		return nil, pipeline.NewErrorWith(types.TeInternal, "invalid partner config", err)
	}

	var password string

	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			password = cred.Value

			break
		}
	}

	var (
		enableActiveMode bool
		activeModeAddr   string
	)

	if c.conf.EnableActiveMode && !partConf.DisableActiveMode {
		port, err := getPortInRange(c.conf.ActiveModeAddress,
			c.conf.ActiveModeMinPort, c.conf.ActiveModeMaxPort)
		if err != nil {
			return nil, err
		}

		enableActiveMode = true
		activeModeAddr = fmt.Sprintf("%s:%d", c.conf.ActiveModeAddress, port)
	}

	addr := conf.GetRealAddress(partner.Address.Host, utils.FormatUint(partner.Address.Port))

	var (
		tlsConfig *tls.Config
		tlsMode   goftp.TLSMode
	)

	if partner.Protocol == "ftps" {
		//nolint:errcheck //error is guaranteed to be nil
		serverName, _, _ := net.SplitHostPort(addr)

		tlsConfig = &tls.Config{
			ServerName: serverName,
			ClientAuth: tls.NoClientCert,
			MinVersion: max(
				c.conf.MinTLSVersion.TLS(),
				partConf.MinTLSVersion.TLS()),
		}

		if auth.AddTLSAuthorities(pip.DB, tlsConfig) != nil {
			return nil, pipeline.NewError(types.TeInternal, "failed to setup the TLS authorities")
		}

		for _, dbCert := range pip.TransCtx.RemoteAccountCreds {
			if dbCert.Type == auth.TLSCertificate {
				cert, err := tls.X509KeyPair([]byte(dbCert.Value), []byte(dbCert.Value2))
				if err != nil {
					pip.Logger.Warningf("failed to parse TLS certificate %q: %v", dbCert.Name, err)

					continue
				}

				tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
			}
		}

		for _, dbCert := range pip.TransCtx.RemoteAgentCreds {
			if dbCert.Type == auth.TLSTrustedCertificate {
				tlsConfig.RootCAs.AppendCertsFromPEM([]byte(dbCert.Value))
			}
		}

		if !partConf.DisableTLSSessionReuse {
			tlsConfig.SessionTicketsDisabled = false
			tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(0)
		}

		if partConf.UseImplicitTLS {
			tlsMode = goftp.TLSImplicit
		}
	}

	ftpConf := goftp.Config{
		Timeout:          clientDefaultConnTimeout,
		User:             account.Login,
		Password:         password,
		TLSConfig:        tlsConfig,
		TLSMode:          tlsMode,
		Logger:           pip.Logger.AsStdLogger(log.LevelTrace).Writer(),
		ActiveTransfers:  enableActiveMode,
		ActiveListenAddr: activeModeAddr,
		DisableEPSV:      partConf.DisableEPSV,
	}

	cli, dialErr := goftp.DialConfig(ftpConf, addr)
	if dialErr != nil {
		return nil, toPipelineError(dialErr, "could not connect to FTP server")
	}

	return cli, nil
}
