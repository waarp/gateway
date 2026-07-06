// Package r66 contains the functions necessary to execute a file transfer
// using the R66 protocol. The package defines both a client and a server.
package r66

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errNoPassword = errors.New("the R66-TLS server is missing a password")

// service represents a r66 service, which encompasses a r66 server usable for
// transfers.
type service struct {
	db      *database.DB
	dbAgent *model.LocalAgent

	logger *log.Logger
	state  utils.State
	tracer func() pipeline.Trace

	r66Conf *ServerConfigTLS
	list    net.Listener
	server  *r66.Server
}

func (s *service) Name() string { return s.dbAgent.Name }

func (s *service) reportError(err error) {
	if err == nil {
		return
	}

	s.logger.Error(err.Error())
	s.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(s.dbAgent.Name, err)
}

func (s *service) makeTLSConf() *tls.Config {
	standardConfig := protoutils.GetServerTLSConfig(s.db, s.logger, s.dbAgent.ID)

	return &tls.Config{
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			if !compatibility.IsLegacyR66CertificateAllowed {
				return standardConfig.GetConfigForClient(chi)
			}

			legacyConfig := &tls.Config{
				MinVersion:   protoutils.GetMinTLSVersion(s.dbAgent.ProtoConfig),
				Certificates: []tls.Certificate{compatibility.LegacyR66Cert},
				ClientAuth:   tls.RequestClientCert,
			}

			if !r66auth.UsesLegacyCert(s.db, s.dbAgent) {
				var err error
				if legacyConfig.Certificates, err = protoutils.GetServerCertificates(
					s.db, s.logger, s.dbAgent); err != nil {
					return nil, err
				}
			}

			return legacyConfig, nil
		},
	}
}

// Start launches a r66 service with an integrated r66 server.
func (s *service) Start() (retErr error) {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.dbAgent.Name)
	defer s.reportError(retErr)

	if err := s.db.Get(s.dbAgent, "id=?", s.dbAgent.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the R66 server: %w", err)
	}

	s.logger = logging.NewLogger(s.dbAgent.Name)
	s.logger.Info("Starting R66 server...")

	if err := s.start(); err != nil {
		return err
	}

	s.logger.Infof("R66 server started successfully on %q", s.list.Addr().String())
	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *service) start() error {
	if err := utils.JSONConvert(s.dbAgent.ProtoConfig, &s.r66Conf); err != nil {
		return fmt.Errorf("failed to parse the R66 proto config: %w", err)
	}

	var pswd model.Credential
	if err := s.db.Get(&pswd, "type=?", auth.Password).And(s.dbAgent.GetCredCond()).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return errNoPassword
		}

		return fmt.Errorf("failed to retrieve the R66 server's password: %w", err)
	}

	login := s.r66Conf.ServerLogin
	if login == "" {
		login = s.dbAgent.Name
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

	return s.listen()
}

func (s *service) listen() error {
	addr := s.db.Config.Overrides.GetRealAddress(s.dbAgent.Address.Host,
		utils.FormatUint(s.dbAgent.Address.Port))

	var listErr error
	if s.list, listErr = protoutils.Listen("tcp", addr); listErr != nil {
		return fmt.Errorf("failed to start R66 listener: %w", listErr)
	}

	if s.dbAgent.Protocol == R66TLS {
		s.list = tls.NewListener(s.list, s.makeTLSConf())
	}

	go func() {
		if err := s.server.Serve(s.list); err != nil {
			s.logger.Errorf("Server stopped unexpectedly: %q", err)
			s.state.Set(utils.StateError, fmt.Sprintf("server stopped unexpectedly: %v", err))
		}
	}()

	return nil
}

// Stop shuts down the r66 server and stops the service.
func (s *service) Stop(ctx context.Context) (retErr error) {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutting down R66 server")
	defer s.reportError(retErr)

	if err := s.stop(ctx); err != nil {
		return err
	}

	s.logger.Info("R66 server shutdown successful")
	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *service) stop(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Go(func() {
		s.logger.Debug("Closing listener...")
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Warningf("Failed to close R66 listener: %v", err)
		}
	})

	if err := pipeline.List.StopAllFromServer(ctx, s.dbAgent.ID); err != nil {
		return fmt.Errorf("failed to interrupt R66 transfers: %w", err)
	}

	wg.Wait()

	return nil
}

// State returns the r66 service's state.
func (s *service) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *service) SetTracer(f func() pipeline.Trace) { s.tracer = f }
