package pesit

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
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

func (s *Service) Name() string { return s.localAgent.Name }

func (s *Service) reportError(err error) {
	if err == nil {
		return
	}

	s.logger.Error(err.Error())
	s.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(s.localAgent.Name, err)
}

func (s *Service) Start() (retErr error) {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.localAgent.Name)
	defer s.reportError(retErr)

	if err := s.db.Get(s.localAgent, "id=?", s.localAgent.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the PeSIT server: %w", err)
	}

	s.logger = logging.NewLogger(s.localAgent.Name)
	s.logger.Info("Starting PeSIT server...")

	if err := utils.JSONConvert(s.localAgent.ProtoConfig, &s.conf); err != nil {
		return fmt.Errorf("failed to parse the pesit agent's proto config: %w", err)
	}

	addr, err := s.listen()
	if err != nil {
		return fmt.Errorf("failed to start the PeSIT server: %w", err)
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("PeSIT server started successfully on %s", addr)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Stopping Pesit service...")

	if err := s.stop(ctx); err != nil {
		s.logger.Errorf("Failed to stop Pesit server: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("Pesit server stopped successfully.")

	return nil
}

func (s *Service) State() (utils.StateCode, string)  { return s.state.Get() }
func (s *Service) SetTracer(f func() pipeline.Trace) { s.tracer = f }
