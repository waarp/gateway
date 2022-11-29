package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

// Service represents an instance of SFTP server.
type Service struct {
	db      *database.DB
	agentID int64
	logger  *log.Logger

	state            state.State
	listener         *sshListener
	runningTransfers *service.TransferMap
}

// NewService returns a new SFTP service instance with the given attributes.
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

func (s *Service) start(agent *model.LocalAgent) error {
	s.agentID = agent.ID

	var protoConfig config.SftpProtoConfig
	if err := json.Unmarshal(agent.ProtoConfig, &protoConfig); err != nil {
		return fmt.Errorf("cannot parse the protocol configuration of this agent: %w", err)
	}

	hostKeys, err := agent.GetCryptos(s.db)
	if err != nil {
		s.logger.Error("Failed to retrieve the server host keys: %s", err)

		return fmt.Errorf("failed to retrieve the server host keys: %w", err)
	}

	sshConf, err1 := getSSHServerConfig(s.db, hostKeys, &protoConfig, agent)
	if err1 != nil {
		s.logger.Error("Failed to parse the SSH server configuration: %s", err1)

		return fmt.Errorf("failed to parse the SSH server configuration: %w", err1)
	}

	addr, err2 := conf.GetRealAddress(agent.Address)
	if err2 != nil {
		s.logger.Error("Failed to indirect the server address: %s", err2)

		return fmt.Errorf("failed to indirect the server address: %w", err2)
	}

	listener, err3 := net.Listen("tcp", addr)
	if err3 != nil {
		s.logger.Error("Failed to start server listener: %s", err3)

		return fmt.Errorf("failed to start server listener: %w", err3)
	}

	s.listener = &sshListener{
		DB:               s.db,
		Logger:           s.logger,
		AgentID:          agent.ID,
		SSHConf:          sshConf,
		Listener:         listener,
		runningTransfers: s.runningTransfers,
		shutdown:         make(chan struct{}),
	}

	s.listener.handlerMaker = s.listener.makeHandlers
	go s.listener.listen()

	return nil
}

// Start starts the SFTP service.
func (s *Service) Start(agent *model.LocalAgent) error {
	if code, _ := s.state.Get(); code != state.Offline && code != state.Error {
		s.logger.Notice("Cannot start the server because it is already running.")

		return nil
	}

	s.logger.Info("Starting SFTP server...")
	s.state.Set(state.Starting, "")

	if err := s.start(agent); err != nil {
		s.state.Set(state.Error, err.Error())
		s.logger.Error("Failed to start SFTP service")

		return err
	}

	s.logger.Info("SFTP server started successfully on %s", s.listener.Listener.Addr().String())
	s.state.Set(state.Running, "")

	return nil
}

// Stop stops the SFTP service.
func (s *Service) Stop(ctx context.Context) error {
	if code, _ := s.State().Get(); code == state.Error || code == state.Offline {
		s.logger.Info("Server is already offline, nothing to do")

		return nil
	}

	s.logger.Info("Shutting down SFTP server")

	s.state.Set(state.ShuttingDown, "")
	defer s.state.Set(state.Offline, "")

	if err := s.listener.close(ctx); err != nil {
		s.logger.Error("Failed to shut down SFTP server, forcing exit")

		return err
	}

	s.logger.Info("SFTP server shutdown successful")

	return nil
}

// State returns the state of the SFTP service.
func (s *Service) State() *state.State {
	return &s.state
}

// ManageTransfers returns a map of the transfers currently running on the server.
func (s *Service) ManageTransfers() *service.TransferMap {
	return s.runningTransfers
}
