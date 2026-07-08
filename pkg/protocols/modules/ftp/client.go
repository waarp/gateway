package ftp

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const clientDefaultConnTimeout = 5 * time.Second // 5s

type client struct {
	db       *database.DB
	dbClient *model.Client
	state    utils.State
	logger   *log.Logger

	conf *ClientConfigTLS
}

func newClient(db *database.DB, dbClient *model.Client) *client {
	c := &client{db: db, dbClient: dbClient}

	return c
}

func (c *client) Name() string { return c.dbClient.Name }

func (c *client) reportError(err error) {
	if err == nil {
		return
	}

	c.logger.Error(err.Error())
	c.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(c.dbClient.Name, err)
}

func (c *client) Start() (retErr error) {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	defer c.reportError(retErr)

	if err := c.db.Get(c.dbClient, "id=?", c.dbClient.ID).Run(); err != nil {
		return fmt.Errorf("failed to get client from database: %w", err)
	}

	c.logger = logging.NewLogger(c.dbClient.Name)
	c.logger.Info("Starting FTP client...")

	c.conf = &ClientConfigTLS{}
	if err := utils.JSONConvert(c.dbClient.ProtoConfig, c.conf); err != nil {
		return fmt.Errorf("invalid client config: %w", err)
	}

	c.state.Set(utils.StateRunning, "")
	c.logger.Info("FTP client started successfully")

	return nil
}

func (c *client) Stop(ctx context.Context) (retErr error) {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	c.logger.Info("Stopping FTP client...")
	defer c.reportError(retErr)

	if err := pipeline.List.StopAllFromClient(ctx, c.dbClient.ID); err != nil {
		return fmt.Errorf("failed to stop the FTP client: %w", err)
	}

	c.state.Set(utils.StateOffline, "")
	c.logger.Info("FTP client stopped successfully")

	return nil
}

func (c *client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	ftpClient, err := connect(pip.Logger, pip.TransCtx, c.conf, pip.DB.Config.Overrides)
	if err != nil {
		return nil, err
	}

	if pip.TransCtx.Rule.IsSend {
		return &clientStorTransfer{client: ftpClient, pip: pip}, nil
	}

	return &clientRetrTransfer{client: ftpClient, pip: pip}, nil
}

func Connect(logger *log.Logger, ctx *model.TransferContext, overrides *conf.ConfigOverride,
) (*goftp.Client, *pipeline.Error) {
	var clientConf ClientConfigTLS
	if err := utils.JSONConvert(ctx.Client.ProtoConfig, &clientConf); err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal, "invalid client config")
	}

	return connect(logger, ctx, &clientConf, overrides)
}

func connect(logger *log.Logger, ctx *model.TransferContext, clientConf *ClientConfigTLS,
	overrides *conf.ConfigOverride,
) (*goftp.Client, *pipeline.Error) {
	partner := ctx.RemoteAgent
	account := ctx.RemoteAccount

	var partConf PartnerConfigTLS
	if err := utils.JSONConvert(partner.ProtoConfig, &partConf); err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal, "invalid partner config")
	}

	var password string

	for _, cred := range ctx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			password = cred.Value

			break
		}
	}

	var (
		enableActiveMode bool
		activeModeAddr   string
	)

	if clientConf.EnableActiveMode && !partConf.DisableActiveMode {
		port, err := getPortInRange(clientConf.ActiveModeAddress,
			clientConf.ActiveModeMinPort, clientConf.ActiveModeMaxPort)
		if err != nil {
			return nil, err
		}

		enableActiveMode = true
		activeModeAddr = fmt.Sprintf("%s:%d", clientConf.ActiveModeAddress, port)
	}

	addr := overrides.GetRealAddress(partner.Address.Host, utils.FormatUint(partner.Address.Port))

	var (
		tlsConfig *tls.Config
		tlsMode   goftp.TLSMode
	)

	if partner.Protocol == FTPS {
		var err *pipeline.Error
		if tlsConfig, tlsMode, err = mkTLSConfig(logger, ctx, &partConf); err != nil {
			return nil, err
		}
	}

	ftpConf := goftp.Config{
		Timeout:          clientDefaultConnTimeout,
		User:             account.Login,
		Password:         password,
		TLSConfig:        tlsConfig,
		TLSMode:          tlsMode,
		Logger:           logger.AsStdLogger(log.LevelTrace).Writer(),
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

func mkTLSConfig(logger *log.Logger, ctx *model.TransferContext, partConf *PartnerConfigTLS,
) (tlsConfig *tls.Config, tlsMode goftp.TLSMode, pErr *pipeline.Error) {
	var tlsErr error
	if tlsConfig, tlsErr = protoutils.GetClientTLSConfig(ctx, logger); tlsErr != nil {
		return nil, 0, pipeline.NewErrorWith(tlsErr, types.TeInternal, "failed to get TLS config")
	}

	tlsConfig.ClientAuth = tls.NoClientCert

	for _, dbCert := range ctx.RemoteAccountCreds {
		if dbCert.Type == auth.TLSCertificate {
			cert, err := tls.X509KeyPair([]byte(dbCert.Value), []byte(dbCert.Value2))
			if err != nil {
				logger.Warningf("failed to parse TLS certificate %q: %v", dbCert.Name, err)

				continue
			}

			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}
	}

	for _, dbCert := range ctx.RemoteAgentCreds {
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

	return tlsConfig, tlsMode, nil
}
