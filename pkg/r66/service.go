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
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
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
	db      *database.DB
	logger  *log.Logger
	agentID int64
	state   state.State

	r66Conf  *config.R66ProtoConfig
	list     net.Listener
	server   *r66.Server
	shutdown chan struct{}

	runningTransfers *service.TransferMap
}

// NewService returns a new R66 service instance with the given attributes.
func NewService(db *database.DB, logger *log.Logger) proto.Service {
	return newService(db, logger)
}

func newService(db *database.DB, logger *log.Logger) *Service {
	return &Service{
		db:               db,
		logger:           logger,
		runningTransfers: service.NewTransferMap(),
	}
}

func (s *Service) makeTLSConf(agent *model.LocalAgent) (*tls.Config, error) {
	certs, err := agent.GetCryptos(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the server's certificates: %w", err)
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
func (s *Service) Start(agent *model.LocalAgent) error {
	if code, _ := s.state.Get(); code != state.Offline && code != state.Error {
		s.logger.Notice("Cannot start the server because it is already running.")

		return nil
	}

	s.agentID = agent.ID

	s.logger.Info("Starting R66 server '%s'...", agent.Name)
	s.state.Set(state.Starting, "")

	s.r66Conf = &config.R66ProtoConfig{}
	if err := json.Unmarshal(agent.ProtoConfig, s.r66Conf); err != nil {
		s.logger.Error("Failed to parse server the R66 proto config: %s", err)
		err1 := fmt.Errorf("failed to parse the R66 proto config: %w", err)
		s.state.Set(state.Error, err1.Error())

		return err1
	}

	pwd, err := utils.AESDecrypt(database.GCM, s.r66Conf.ServerPassword)
	if err != nil {
		s.logger.Error("Failed to decrypt server password: %s", err)
		dErr := fmt.Errorf("failed to decrypt server password: %w", err)
		s.state.Set(state.Error, dErr.Error())

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

	if err := s.listen(agent); err != nil {
		s.state.Set(state.Error, err.Error())

		return err
	}

	s.shutdown = make(chan struct{})

	s.state.Set(state.Running, "")
	s.logger.Info("R66 server started at %s", agent.Address)

	return nil
}

func (s *Service) listen(agent *model.LocalAgent) error {
	addr, err := conf.GetRealAddress(agent.Address)
	if err != nil {
		s.logger.Error("Failed to parse server TLS config: %s", err)

		return fmt.Errorf("failed to indirect the server address: %w", err)
	}

	if agent.Protocol == ProtocolR66TLS {
		tlsConf, err2 := s.makeTLSConf(agent)
		if err2 != nil {
			s.logger.Error("Failed to parse server TLS config: %s", err2)
			s.state.Set(state.Error, "failed to parse server TLS config")

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
				s.state.Set(state.Error, fmt.Sprintf("server stopped unexpectedly: %s", err))
			}
		}
	}()

	return nil
}

// Stop shuts down the r66 server and stops the service.
func (s *Service) Stop(ctx context.Context) error {
	if code, _ := s.State().Get(); code == state.Error || code == state.Offline {
		s.logger.Notice("Server is already offline, nothing to do")

		return nil
	}

	s.logger.Info("Shutting down R66 server")

	s.state.Set(state.ShuttingDown, "")
	defer s.state.Set(state.Offline, "")

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
func (s *Service) State() *state.State {
	return &s.state
}

// ManageTransfers returns a map of the transfers currently running on the server.
func (s *Service) ManageTransfers() *service.TransferMap {
	return s.runningTransfers
}
