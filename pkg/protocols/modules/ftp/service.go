package ftp

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/log"
	ftplib "github.com/fclairamb/ftpserverlib"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type service struct {
	db    *database.DB
	agent *model.LocalAgent

	logger  *log.Logger
	server  *ftplib.FtpServer
	handler *handler
	state   utils.State
}

func newServer(db *database.DB, agent *model.LocalAgent) *service {
	return &service{
		db:     db,
		agent:  agent,
		logger: logging.NewLogger(agent.Name),
	}
}

func (s *service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger.Info("Starting FTP server...")

	if err := s.start(); err != nil {
		s.logger.Error("Failed to start FTP server: %s", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Info("FTP server started successfully on %s", s.server.Addr())

	return nil
}

func (s *service) start() error {
	var serverConf ServerConfigTLS
	if err := utils.JSONConvert(s.agent.ProtoConfig, &serverConf); err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}

	s.handler = &handler{
		db:         s.db,
		logger:     s.logger,
		dbServer:   s.agent,
		serverConf: &serverConf,
	}

	if s.agent.Protocol == FTPS {
		s.handler.mkTLSConfig()
	}

	s.server = ftplib.NewFtpServer(s.handler)
	s.server.Logger = &ftpServerLogger{s.logger}

	if err := s.server.Listen(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	go func() {
		if err := s.server.Serve(); err != nil {
			_ = s.server.Stop() //nolint:errcheck // error does not matter at this point

			s.logger.Error("Failed to serve FTP server: %s", err)
			s.state.Set(utils.StateError, err.Error())

			return
		}
	}()

	return nil
}

func (s *service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Stopping FTP server...")

	if err := s.stop(ctx); err != nil {
		s.logger.Error("Failed to stop FTP server: %s", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("FTP server stopped")

	return nil
}

func (s *service) stop(ctx context.Context) error {
	s.logger.Info("Stopping FTP server...")

	if err := pipeline.List.StopAllFromServer(ctx, s.agent.ID); err != nil {
		_ = s.server.Stop() //nolint:errcheck // error does not matter at this point

		return fmt.Errorf("failed to stop running transfers: %w", err)
	}

	if err := s.server.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}

func (s *service) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *service) SetTracer(tracer func() pipeline.Trace) { s.handler.tracer = tracer }
