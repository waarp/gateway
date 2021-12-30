// Package r66 contains the functions necessary to execute a file transfer
// using the R66 protocol. The package defines both a client and a server.
package r66

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

const (
	ProtocolR66    = config.ProtocolR66
	ProtocolR66TLS = config.ProtocolR66TLS
)

var errNoCertificates = errors.New("the R66-TLS server is missing a certificate")

// Service represents a r66 service, which encompasses a r66 server usable for
// transfers.
type Service struct {
	db     *database.DB
	logger *log.Logger
	agent  *model.LocalAgent
	state  *service.State

	r66Conf  *config.R66ProtoConfig
	list     net.Listener
	server   *r66.Server
	shutdown chan struct{}

	runningTransfers *service.TransferMap
}

// NewService returns a new R66 service instance with the given attributes.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.ProtoService {
	return newService(db, agent, logger)
}

func newService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) *Service {
	return &Service{
		db:               db,
		agent:            agent,
		logger:           logger,
		state:            &service.State{},
		shutdown:         make(chan struct{}),
		runningTransfers: service.NewTransferMap(),
	}
}

func (s *Service) makeTLSConf() (*tls.Config, error) {
	certs, err := s.agent.GetCryptos(s.db)
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 {
		return nil, errNoCertificates
	}

	tlsCerts := make([]tls.Certificate, len(certs))

	for i := range certs {
		var err error
		tlsCerts[i], err = tls.X509KeyPair(
			[]byte(certs[i].Certificate),
			[]byte(certs[i].PrivateKey),
		)

		if err != nil {
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
func (s *Service) Start() error {
	s.logger.Info("Starting R66 server '%s'...", s.agent.Name)
	s.state.Set(service.Starting, "")

	s.r66Conf = &config.R66ProtoConfig{}
	if err := json.Unmarshal(s.agent.ProtoConfig, s.r66Conf); err != nil {
		s.logger.Error("Failed to parse server ProtoConfig: %s", err)
		err1 := fmt.Errorf("failed to parse ProtoConfig: %w", err)
		s.state.Set(service.Error, err1.Error())

		return err1
	}

	pwd, err := utils.AESDecrypt(database.GCM, s.r66Conf.ServerPassword)
	if err != nil {
		s.logger.Error("Failed to decrypt server password: %s", err)
		dErr := fmt.Errorf("failed to decrypt server password: %w", err)
		s.state.Set(service.Error, dErr.Error())

		return dErr
	}

	s.server = &r66.Server{
		Login:    s.r66Conf.ServerLogin,
		Password: []byte(pwd),
		Logger:   s.logger.AsStdLogger(log.LevelTrace),
		Conf: &r66.Config{
			FileSize:   true,
			FinalHash:  !s.r66Conf.NoFinalHash,
			DigestAlgo: "SHA-256",
			Proxified:  false,
		},
		Handler: &authHandler{Service: s},
	}

	if err := s.listen(); err != nil {
		s.state.Set(service.Error, err.Error())

		return err
	}

	s.state.Set(service.Running, "")
	s.logger.Info("R66 server started at %s", s.agent.Address)

	return nil
}

func (s *Service) listen() error {
	addr, err := conf.GetRealAddress(s.agent.Address)
	if err != nil {
		s.logger.Error("Failed to parse server TLS config: %s", err)

		return fmt.Errorf("failed to indirect the server address: %w", err)
	}

	if s.agent.Protocol == ProtocolR66TLS {
		tlsConf, err2 := s.makeTLSConf()
		if err2 != nil {
			s.logger.Error("Failed to parse server TLS config: %s", err2)
			s.state.Set(service.Error, "failed to parse server TLS config")

			return err2
		}

		s.list, err = tls.Listen("tcp", addr, tlsConf)
	} else {
		s.list, err = net.Listen("tcp", addr)
	}

	if err != nil {
		s.logger.Error("Failed to start R66 listener: %s", err)

		return fmt.Errorf("failed to start R66 listener: %w", err)
	}

	go func() {
		if err := s.server.Serve(s.list); err != nil {
			select {
			case <-s.shutdown:
				return

			default:
				s.logger.Error("Server stopped unexpectedly: %s", err)
				s.state.Set(service.Error, fmt.Sprintf("server stopped unexpectedly: %s", err))
			}
		}
	}()

	return nil
}

// Stop shuts down the r66 server and stops the service.
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down R66 server")

	if code, _ := s.State().Get(); code == service.Error || code == service.Offline {
		s.logger.Info("Server is already offline, nothing to do")

		return nil
	}

	s.state.Set(service.ShuttingDown, "")
	defer s.state.Set(service.Offline, "")

	if s.shutdown == nil {
		s.shutdown = make(chan struct{})
	}

	close(s.shutdown)

	if err := s.runningTransfers.InterruptAll(ctx); err != nil {
		s.logger.Error("Failed to interrupt R66 transfers, forcing exit")

		return fmt.Errorf("failed to interrupt R66 transfers: %w", err)
	}

	s.logger.Debug("Closing listener...")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Warning("Failed to properly shutdown R66 server")
	}

	s.logger.Info("R66 server shutdown successful")

	return nil
}

// State returns the r66 service's state.
func (s *Service) State() *service.State {
	if s.server == nil {
		if code, _ := s.state.Get(); code == service.Running {
			s.state.Set(service.Offline, "")
		}
	}

	return s.state
}

// ManageTransfers returns a map of the transfers currently running on the server.
func (s *Service) ManageTransfers() *service.TransferMap {
	return s.runningTransfers
}
