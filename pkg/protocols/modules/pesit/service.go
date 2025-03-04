package pesit

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Service struct {
	*server
}

func newService(db *database.DB, serv *model.LocalAgent) *Service {
	return &Service{
		server: &server{
			db:         db,
			localAgent: serv,
			logger:     logging.NewLogger(serv.Name),
		},
	}
}

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger.Info("Starting Pesit server...")

	addr, err := s.start()
	if err != nil {
		s.logger.Error("Failed to start Pesit server: %s", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Info("Pesit server started successfully on %s", addr)

	return nil
}

func (s *Service) start() (string, error) {
	var conf ServerConfigTLS
	if err := utils.JSONConvert(s.localAgent.ProtoConfig, &conf); err != nil {
		return "", fmt.Errorf("failed to parse the pesit agent's proto config: %w", err)
	}

	s.server.conf = &conf

	return s.server.listen()
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Stopping Pesit service...")

	if err := s.server.stop(ctx); err != nil {
		s.logger.Error("Failed to stop Pesit server: %s", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("Pesit server stopped successfully.")

	return nil
}

func (s *Service) State() (utils.StateCode, string)  { return s.state.Get() }
func (s *Service) SetTracer(f func() pipeline.Trace) { s.tracer = f }
