package sftp

import (
	"context"
	"fmt"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// Server represents an instance of SFTP server.
type Server struct {
	db     *database.Db
	agent  *model.LocalAgent
	logger *log.Logger

	state    service.State
	listener *sshServer
}

// NewServer returns a new SFTP server instance with the given attributes.
func NewServer(db *database.Db, agent *model.LocalAgent, logger *log.Logger) *Server {
	return &Server{
		db:     db,
		agent:  agent,
		logger: logger,
	}
}

// Start starts the SFTP server.
func (s *Server) Start() error {
	start := func() error {
		cert, err := loadCert(s.db, s.agent)
		if err != nil {
			return err
		}

		sshConf, err := loadSSHConfig(s.db, cert)
		if err != nil {
			return err
		}

		addr, port, err := parseServerAddr(s.agent)
		if err != nil {
			return err
		}

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%v", addr, port))
		if err != nil {
			return err
		}

		s.listener = newListener(s.db, s.logger, listener, s.agent, sshConf)
		s.listener.listen()
		return nil
	}

	s.logger.Info("Starting SFTP server...")
	s.state.Set(service.Starting, "")

	if err := start(); err != nil {
		s.state.Set(service.Error, err.Error())
		s.logger.Infof("Failed to start SFTP server: %s", err)
		return err
	}

	s.logger.Info("SFTP server started successfully")
	s.state.Set(service.Running, "")
	return nil
}

// Stop stops the SFTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Infof("Shutting down SFTP service '%s'", s.agent.Name)
	if s.listener.close(ctx) != nil {
		s.logger.Errorf("Failed to shut SFTP server down, forcing exit")
	} else {
		s.logger.Info("SFTP server shutdown successful")
	}
	s.state.Set(service.Offline, "")
	return nil
}

// State returns the state of the SFTP server.
func (s *Server) State() *service.State {
	return &s.state
}
