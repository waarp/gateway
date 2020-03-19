package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// Service represents an instance of SFTP server.
type Service struct {
	db     *database.DB
	agent  *model.LocalAgent
	logger *log.Logger

	state    service.State
	listener *sshListener
}

// NewService returns a new SFTP service instance with the given attributes.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) *Service {
	return &Service{
		db:     db,
		agent:  agent,
		logger: logger,
	}
}

// Start starts the SFTP service.
func (s *Service) Start() error {
	start := func() error {
		var protoConfig config.SftpProtoConfig
		if err := json.Unmarshal(s.agent.ProtoConfig, &protoConfig); err != nil {
			return err
		}

		cert, err := loadCert(s.db, s.agent)
		if err != nil {
			return err
		}

		sshConf, err := gertSSHServerConfig(s.db, cert, &protoConfig)
		if err != nil {
			return err
		}

		addr, port, err := parseServerAddr(s.agent)
		if err != nil {
			return err
		}

		listener, err := net.Listen("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
		if err != nil {
			return err
		}

		s.listener = &sshListener{
			DB:           s.db,
			Logger:       s.logger,
			Agent:        s.agent,
			ServerConfig: sshConf,
			ProtoConfig:  &protoConfig,
			Listener:     listener,
		}
		s.listener.ctx, s.listener.cancel = context.WithCancel(context.Background())
		s.listener.listen()
		return nil
	}

	s.logger.Infof("Starting SFTP server...")
	s.state.Set(service.Starting, "")

	if err := start(); err != nil {
		s.state.Set(service.Error, err.Error())
		s.logger.Infof("Failed to start SFTP service: %s", err)
		return err
	}

	s.logger.Infof("SFTP server started successfully on %s", s.listener.Listener.Addr().String())
	s.state.Set(service.Running, "")
	return nil
}

// Stop stops the SFTP service.
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Infof("Shutting down SFTP service '%s'", s.agent.Name)
	if s.listener.close(ctx) != nil {
		s.logger.Errorf("Failed to shut SFTP server down, forcing exit")
	} else {
		s.logger.Info("SFTP server shutdown successful")
	}
	s.state.Set(service.Offline, "")
	return nil
}

// State returns the state of the SFTP service.
func (s *Service) State() *service.State {
	return &s.state
}
