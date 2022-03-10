package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// Service represents an instance of SFTP server.
type Service struct {
	db     *database.DB
	agent  *model.LocalAgent
	logger *log.Logger

	state    *service.State
	listener *sshListener
}

// NewService returns a new SFTP service instance with the given attributes.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.ProtoService {
	return newService(db, agent, logger)
}

func newService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) *Service {
	return &Service{
		db:     db,
		agent:  agent,
		logger: logger,
		state:  &service.State{},
	}
}

func (s *Service) start() error {
	var protoConfig config.SftpProtoConfig
	if err := json.Unmarshal(s.agent.ProtoConfig, &protoConfig); err != nil {
		return fmt.Errorf("cannot parse the protocol configuration of this agent: %w", err)
	}

	hostKeys, err := s.agent.GetCryptos(s.db)
	if err != nil {
		s.logger.Errorf("Failed to retrieve the server host keys: %s", err)

		return err
	}

	sshConf, err1 := getSSHServerConfig(s.db, hostKeys, &protoConfig, s.agent)
	if err1 != nil {
		s.logger.Errorf("Failed to parse the SSH server configuration: %s", err1)

		return fmt.Errorf("failed to parse the SSH server configuration: %w", err1)
	}

	addr, err2 := conf.GetRealAddress(s.agent.Address)
	if err2 != nil {
		s.logger.Errorf("Failed to indirect the server address: %s", err2)

		return fmt.Errorf("failed to indirect the server address: %w", err2)
	}

	listener, err3 := net.Listen("tcp", addr)
	if err3 != nil {
		s.logger.Errorf("Failed to start server listener: %s", err3)

		return fmt.Errorf("failed to start server listener: %w", err3)
	}

	s.listener = &sshListener{
		DB:               s.db,
		Logger:           s.logger,
		Agent:            s.agent,
		ProtoConfig:      &protoConfig,
		SSHConf:          sshConf,
		Listener:         listener,
		runningTransfers: service.NewTransferMap(),
		shutdown:         make(chan struct{}),
	}
	s.listener.listen()

	return nil
}

// Start starts the SFTP service.
func (s *Service) Start() error {
	s.logger.Infof("Starting SFTP server...")
	s.state.Set(service.Starting, "")

	if err := s.start(); err != nil {
		s.state.Set(service.Error, err.Error())
		s.logger.Error("Failed to start SFTP service")

		return err
	}

	s.logger.Infof("SFTP server started successfully on %s", s.listener.Listener.Addr().String())
	s.state.Set(service.Running, "")

	return nil
}

// Stop stops the SFTP service.
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down SFTP server")

	if code, _ := s.State().Get(); code == service.Error || code == service.Offline {
		s.logger.Info("Server is already offline, nothing to do")

		return nil
	}

	s.state.Set(service.ShuttingDown, "")
	defer s.state.Set(service.Offline, "")

	if err := s.listener.close(ctx); err != nil {
		s.logger.Error("Failed to shut down SFTP server, forcing exit")

		return err
	}

	s.logger.Info("SFTP server shutdown successful")

	return nil
}

// State returns the state of the SFTP service.
func (s *Service) State() *service.State {
	return s.state
}

// ManageTransfers returns a map of the transfers currently running on the server.
func (s *Service) ManageTransfers() *service.TransferMap {
	return s.listener.runningTransfers
}
