// Package r66 contains the functions necessary to execute a file transfer
// using the R66 protocol. The package defines both a client and a server.
package r66

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errNoCertificates = errors.New("the R66-TLS server is missing a certificate")

// service represents a r66 service, which encompasses a r66 server usable for
// transfers.
type service struct {
	db    *database.DB
	agent *model.LocalAgent

	logger *log.Logger
	state  utils.State
	tracer func() pipeline.Trace

	r66Conf *serverConfig
	list    net.Listener
	server  *r66.Server
}

func (s *service) makeTLSConf(*tls.ClientHelloInfo) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		ClientAuth:       tls.RequestClientCert,
		VerifyConnection: compatibility.LogSha1(s.logger),
	}

	if compatibility.IsLegacyR66CertificateAllowed {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errMissingCertificate
			}

			chain, parsErr := auth.ParseRawCertChain(rawCerts)
			if parsErr != nil {
				return fmt.Errorf("failed to parse the certification chain: %w", parsErr)
			}

			if !compatibility.IsLegacyR66Cert(chain[0]) {
				return auth.VerifyClientCert(s.db, s.logger, s.agent)(rawCerts, nil)
			}

			return nil
		}
	} else {
		tlsConfig.VerifyPeerCertificate = auth.VerifyClientCert(s.db, s.logger, s.agent)
	}

	if usesLegacyCert(s.db, s.agent) {
		tlsConfig.Certificates = []tls.Certificate{compatibility.LegacyR66Cert}
	} else {
		certs, dbErr := s.agent.GetCredentials(s.db, auth.TLSCertificate)
		if dbErr != nil {
			return nil, fmt.Errorf("failed to retrieve the server's certificates: %w", dbErr)
		}

		if len(certs) == 0 {
			return nil, errNoCertificates
		}

		tlsConfig.Certificates = make([]tls.Certificate, len(certs))

		for i, cert := range certs {
			var parseErr error
			tlsConfig.Certificates[i], parseErr = utils.X509KeyPair(cert.Value, cert.Value2)

			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse certificate %s: %w", certs[i].Name, parseErr)
			}
		}
	}

	return tlsConfig, nil
}

// Start launches a r66 service with an integrated r66 server.
func (s *service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.start(); err != nil {
		s.logger.Error("Failed to start R66 service: %s", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *service) start() error {
	s.logger = logging.NewLogger(s.agent.Name)
	s.logger.Info("Starting R66 server...")

	setLogAndReturnError := func(msg string, args ...any) error {
		//nolint:goerr113 //dynamic error is better here for readability
		err := fmt.Errorf(msg, args...)
		s.logger.Error(err.Error())
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.r66Conf = &serverConfig{}
	if err := utils.JSONConvert(s.agent.ProtoConfig, s.r66Conf); err != nil {
		return setLogAndReturnError("Failed to parse the R66 proto config: %s", err)
	}

	var pswd model.Credential
	if err := s.db.Get(&pswd, "type=?", auth.Password).And(s.agent.GetCredCond()).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return setLogAndReturnError("The R66 server is missing a password")
		}

		return setLogAndReturnError("Failed to retrieve the R66 server's password: %w", err)
	}

	login := s.r66Conf.ServerLogin
	if login == "" {
		login = s.agent.Name
	}

	s.server = &r66.Server{
		Login:    login,
		Password: []byte(pswd.Value),
		Logger:   s.logger.AsStdLogger(log.LevelTrace),
		Conf: &r66.Config{
			FileSize:   true,
			FinalHash:  !s.r66Conf.NoFinalHash,
			DigestAlgo: "SHA-256",
			Proxified:  false,
		},
		Handler: &authHandler{service: s},
	}

	if err := s.listen(); err != nil {
		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Info("R66 server started successfully on %s", s.list.Addr().String())

	return nil
}

func (s *service) listen() error {
	addr := conf.GetRealAddress(s.agent.Address.Host,
		utils.FormatUint(s.agent.Address.Port))

	var err error

	if s.agent.Protocol == R66TLS {
		s.list, err = tls.Listen("tcp", addr, &tls.Config{
			MinVersion:         tls.VersionTLS12,
			GetConfigForClient: s.makeTLSConf,
		})
	} else {
		s.list, err = net.Listen("tcp", addr)
	}

	if err != nil {
		s.logger.Error("Failed to start R66 listener: %s", err)

		return fmt.Errorf("failed to start R66 listener: %w", err)
	}

	s.list = &protoutils.TraceListener{Listener: s.list}

	go func() {
		if err := s.server.Serve(s.list); err != nil {
			s.logger.Error("Server stopped unexpectedly: %s", err)
			s.state.Set(utils.StateError, fmt.Sprintf("server stopped unexpectedly: %s", err))
		}
	}()

	return nil
}

// Stop shuts down the r66 server and stops the service.
func (s *service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := s.stop(ctx); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *service) stop(ctx context.Context) error {
	s.logger.Info("Shutting down R66 server")

	var stopErr error

	if err := pipeline.List.StopAllFromServer(ctx, s.agent.ID); err != nil {
		s.logger.Error("Failed to interrupt R66 transfers, forcing exit")

		stopErr = fmt.Errorf("failed to interrupt R66 transfers: %w", err)
	}

	s.logger.Debug("Closing listener...")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Warning("Failed to properly shutdown R66 server: %v", err)
	}

	if stopErr != nil {
		return stopErr
	}

	s.logger.Info("R66 server shutdown successful")

	return nil
}

// State returns the r66 service's state.
func (s *service) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *service) SetTracer(f func() pipeline.Trace) { s.tracer = f }
