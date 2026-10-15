package ftp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	ftplib "github.com/fclairamb/ftpserverlib"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/throttle"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const (
	serverDefaultConnTimeout = 5000 // 5s
	serverDefaultIdleTimeout = 300  // 5min
)

type handler struct {
	db     *database.DB
	logger *log.Logger

	tracer     func() pipeline.Trace
	dbServer   *model.LocalAgent
	serverConf *ServerConfigTLS
	tlsConfig  *tls.Config
}

func (h *handler) getBanner() string {
	return fmt.Sprintf("Welcome to the Waarp-Gateway FTP server %q version %s",
		h.dbServer.Name, version.Num)
}

func (h *handler) GetSettings() (*ftplib.Settings, error) {
	var pasvPortRange *ftplib.PortRange

	if !h.serverConf.DisablePassiveMode {
		pasvPortRange = &ftplib.PortRange{
			Start: int(h.serverConf.PassiveModeMinPort),
			End:   int(h.serverConf.PassiveModeMaxPort),
		}
	}

	addr := h.db.Config.Overrides.GetRealAddress(h.dbServer.Address.Host,
		utils.FormatUint(h.dbServer.Address.Port))

	return &ftplib.Settings{
		ListenAddr:               addr,
		PassiveTransferPortRange: pasvPortRange,
		ActiveTransferPortNon20:  true, // maybe make it configurable ?
		IdleTimeout:              serverDefaultIdleTimeout,
		ConnectionTimeout:        serverDefaultConnTimeout,
		Banner:                   h.getBanner(),
		TLSRequired:              h.serverConf.TLSRequirement.toLib(),
		DisableActiveMode:        h.serverConf.DisableActiveMode,
		DisableSite:              true,
		DisableMFMT:              true,
		EnableHASH:               false, // maybe make configurable ?
		EnableCOMB:               false, // proprietary feature, might enable if requested by users
		DefaultTransferType:      ftplib.TransferTypeBinary,
		ActiveConnectionsCheck:   ftplib.IPMatchRequired,
		PasvConnectionsCheck:     ftplib.IPMatchRequired,
	}, nil
}

func (h *handler) WrapPassiveListener(listener net.Listener) (net.Listener, error) {
	if h.serverConf.DisablePassiveMode {
		//nolint:err113 //too specific
		return nil, errors.New("passive mode is disabled on this server")
	}

	return listener, nil
}

func (h *handler) attempt(cc ftplib.ClientContext, user string) throttle.Attempt {
	return throttle.Default().Attempt(&h.db.Config.Security, FTP+":"+h.dbServer.Name,
		user, cc.RemoteAddr().String())
}

func (h *handler) ClientConnected(ftplib.ClientContext) (string, error) {
	h.logger.Debug("Server control connection opened")
	analytics.AddIncomingConnection()

	return h.getBanner(), nil
}

func (h *handler) ClientDisconnected(ftplib.ClientContext) {
	h.logger.Debug("Server control connection closed")
	analytics.SubIncomingConnection()
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (h *handler) PreAuthUser(cc ftplib.ClientContext, user string) error {
	acc, err := h.dbServer.GetAccount(h.db, user)
	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("Failed to retrieve TLS account: %v", err)

		return errors.New("internal authentication error")
	}

	if acc == nil {
		acc = &model.LocalAccount{}
	}

	cc.SetExtra(acc)

	return nil
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (h *handler) AuthUser(cc ftplib.ClientContext, user, pass string) (ftplib.ClientDriver, error) {
	h.logger.Debugf("Received authentication request from account %q", user)

	acc, ok := cc.Extra().(*model.LocalAccount)
	if !ok {
		return nil, errors.New("internal authentication error")
	}

	attempt := h.attempt(cc, user)
	if attempt.Locked() {
		h.logger.Warningf("Authentication of account %q from %q refused: too many failed attempts",
			user, cc.RemoteAddr())

		return nil, errors.New("invalid credentials")
	}

	// A disallowed address is refused like invalid credentials, so that the
	// answer does not tell whether the password was valid. The actual reason
	// is logged by CheckIP.
	if success, err := protoutils.PasswordAuthentication(h.db, h.logger, acc, pass); err != nil {
		return nil, errors.New("internal authentication error")
	} else if !success || !acc.CheckIP(h.logger, cc.RemoteAddr().String()) {
		if attempt.Fail() {
			h.logger.Warningf("Too many failed authentications for account %q from %q, "+
				"new attempts are temporarily refused", user, cc.RemoteAddr())
		}

		return nil, errors.New("invalid credentials")
	}

	attempt.Succeed()

	h.logger.Debugf("Account %q authenticated successfully", user)

	return &serverFS{
		db:     h.db,
		logger: h.logger,
		tracer: h.tracer,
		dbAcc:  acc,
	}, nil
}

func (h *handler) GetTLSConfig() (*tls.Config, error) {
	if h.dbServer.Protocol != FTPS {
		//nolint:err113 //too specific
		return nil, errors.New("cannot create TLS config for non-FTPS server")
	}

	return h.tlsConfig, nil
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (h *handler) VerifyConnection(cctx ftplib.ClientContext, user string,
	tlsConn *tls.Conn,
) (ftplib.ClientDriver, error) {
	h.logger.Debugf("Received authentication request from account %q", user)

	acc, ok := cctx.Extra().(*model.LocalAccount)
	if !ok {
		return nil, errors.New("internal authentication error")
	}

	// A locked attempt is handled like a refused certificate. A refused
	// certificate is not counted as a failure: the client then falls back to
	// its password, which is counted if invalid.
	attempt := h.attempt(cctx, user)
	if attempt.Locked() {
		//nolint:nilnil //returning "nil, nil" here is required by the interface's definition
		return nil, nil
	}

	state := tlsConn.ConnectionState()

	// A disallowed address is handled like a refused certificate (the client
	// falls back to password authentication, refused as well), so that the
	// answer does not tell whether the certificate was valid.
	success, err := protoutils.CertificateAuthentication(h.db, h.logger, acc, &state)
	if err != nil {
		return nil, errors.New("internal authentication error")
	} else if !success || !acc.CheckIP(h.logger, cctx.RemoteAddr().String()) {
		//nolint:nilnil //returning "nil, nil" here is required by the interface's definition
		return nil, nil
	}

	attempt.Succeed()

	h.logger.Debugf("Account %q authenticated successfully", user)

	return &serverFS{
		db:     h.db,
		logger: h.logger,
		tracer: h.tracer,
		dbAcc:  acc,
	}, nil
}
