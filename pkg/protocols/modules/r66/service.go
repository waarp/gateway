// Package r66 contains the functions necessary to execute a file transfer
// using the R66 protocol. The package defines both a client and a server.
package r66

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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
	certs, err := s.agent.GetCryptos(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the server's certificates: %w", err)
	}

	if len(certs) == 0 {
		return nil, errNoCertificates
	}

	tlsCerts := make([]tls.Certificate, len(certs))

	for i := range certs {
		var err error
		if tlsCerts[i], err = tls.X509KeyPair(
			[]byte(certs[i].Certificate),
			[]byte(certs[i].PrivateKey),
		); err != nil {
			return nil, fmt.Errorf("failed to parse certificate %s: %w", certs[i].Name, err)
		}
	}

	return &tls.Config{
		Certificates:     tlsCerts,
		MinVersion:       tls.VersionTLS12,
		ClientAuth:       tls.RequestClientCert,
		VerifyConnection: compatibility.LogSha1(s.logger),
	}, nil
}

// Start launches a r66 service with an integrated r66 server.
func (s *service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.start(); err != nil {
		s.logger.Error("Failed to start R66 service: %s", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *service) start() error {
	s.logger = logging.NewLogger(s.agent.Name)
	s.logger.Info("Starting R66 server '%s'...", s.agent.Name)

	s.r66Conf = &serverConfig{}
	if err := utils.JSONConvert(s.agent.ProtoConfig, s.r66Conf); err != nil {
		s.logger.Error("Failed to parse server the R66 proto config: %s", err)
		err1 := fmt.Errorf("failed to parse the R66 proto config: %w", err)
		s.state.Set(utils.StateError, err1.Error())

		return err1
	}

	pwd, pwdErr := utils.AESDecrypt(database.GCM, s.r66Conf.ServerPassword)
	if pwdErr != nil {
		s.logger.Error("Failed to decrypt server password: %s", pwdErr)
		dErr := fmt.Errorf("failed to decrypt server password: %w", pwdErr)
		s.state.Set(utils.StateError, dErr.Error())

		return dErr
	}

	login := s.r66Conf.ServerLogin
	if login == "" {
		login = s.agent.Name
	}

	s.server = &r66.Server{
		Login:    login,
		Password: []byte(pwd),
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
	s.logger.Info("R66 server started at %s", s.agent.Address)

	return nil
}

func (s *service) listen() error {
	addr, err := conf.GetRealAddress(s.agent.Address)
	if err != nil {
		s.logger.Error("Failed to parse server TLS config: %s", err)

		return fmt.Errorf("failed to indirect the server address: %w", err)
	}

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
