package ftp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"
	ftplib "github.com/fclairamb/ftpserverlib"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
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
		rangeStart := int(h.serverConf.PassiveModeMinPort)
		rangeEnd := int(h.serverConf.PassiveModeMaxPort)

		if rangeStart != 0 || rangeEnd != 0 {
			pasvPortRange = &ftplib.PortRange{
				Start: rangeStart,
				End:   rangeEnd,
			}
		}
	}

	return &ftplib.Settings{
		ListenAddr:               h.dbServer.Address.String(),
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
		//nolint:goerr113 //too specific
		return nil, errors.New("passive mode is disabled on this server")
	}

	return listener, nil
}

func (h *handler) ClientConnected(ftplib.ClientContext) (string, error) {
	return h.getBanner(), nil
}

func (h *handler) ClientDisconnected(ftplib.ClientContext) {}

//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
func (h *handler) AuthUser(_ ftplib.ClientContext, user, pass string) (ftplib.ClientDriver, error) {
	h.logger.Debug("Received authentication request from account %q", user)

	var acc model.LocalAccount
	if err := h.db.Get(&acc, "local_agent_id=? AND login=?", h.dbServer.ID, user).
		Run(); err != nil && !database.IsNotFound(err) {
		h.logger.Error("Failed to retrieve account: %s", err)

		return nil, errors.New("internal authentication error")
	}

	if res, err := acc.Authenticate(h.db, h.dbServer, auth.Password, pass); err != nil {
		h.logger.Error("Failed to authenticate account %q: %s", user, err)

		return nil, errors.New("internal authentication error")
	} else if !res.Success {
		h.logger.Warning("Invalid credentials for account %q: %s", user, res.Reason)

		return nil, errors.New("invalid credentials")
	}

	h.logger.Debug("Account %q authenticated successfully", user)

	return &serverFS{
		db:       h.db,
		logger:   h.logger,
		tracer:   h.tracer,
		dbServer: h.dbServer,
		dbAcc:    &acc,
	}, nil
}

func (h *handler) GetTLSConfig() (*tls.Config, error) {
	if h.dbServer.Protocol != FTPS {
		//nolint:goerr113 //too specific
		return nil, errors.New("cannot create TLS config for non-FTPS server")
	}

	return h.tlsConfig, nil
}

func (h *handler) mkTLSConfig() {
	//nolint:gosec //TLS version is set by the user
	h.tlsConfig = &tls.Config{
		MinVersion: protoutils.ParseTLSVersion(h.serverConf.MinTLSVersion),
		ClientAuth: tls.RequestClientCert,
		//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			dbCerts, dbErr := h.dbServer.GetCredentials(h.db, auth.TLSCertificate)
			if dbErr != nil {
				h.logger.Error("Failed to retrieve TLS certificates: %v", dbErr)

				return nil, errors.New("internal database error")
			}

			for _, dbCert := range dbCerts {
				cert, err := tls.X509KeyPair([]byte(dbCert.Value), []byte(dbCert.Value2))
				if err != nil {
					h.logger.Warning("Failed to parse TLS certificate: %v", err)
				}

				if chi.SupportsCertificate(&cert) == nil {
					return &cert, nil
				}
			}

			return nil, errors.New("no valid TLS certificate found")
		},
	}
}

//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
func (h *handler) VerifyConnection(_ ftplib.ClientContext, user string,
	tlsConn *tls.Conn,
) (ftplib.ClientDriver, error) {
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		//nolint:nilnil //returning "nil, nil" here is required by the interface's definition
		return nil, nil
	}

	var acc model.LocalAccount
	if err := h.db.Get(&acc, "local_agent_id=? AND login=?", h.dbServer.ID,
		user).Run(); err != nil && !database.IsNotFound(err) {
		h.logger.Error("Failed to retrieve TLS account: %v", err)

		return nil, errors.New("internal authentication error")
	}

	res, err := acc.Authenticate(h.db, h.dbServer, auth.TLSTrustedCertificate, certs)
	if err != nil {
		h.logger.Error("Failed to authenticate account %q with TLS: %s", user, err)

		return nil, errors.New("internal authentication error")
	} else if !res.Success {
		h.logger.Warning("Invalid credentials for account %q: %s", user, res.Reason)

		return nil, errors.New("invalid credentials")
	}

	return &serverFS{
		db:       h.db,
		logger:   h.logger,
		tracer:   h.tracer,
		dbServer: h.dbServer,
		dbAcc:    &acc,
	}, nil
}
