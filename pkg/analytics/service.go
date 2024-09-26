package analytics

import (
	"context"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ServiceName = "Analytics"

//nolint:gochecknoglobals //global var is better for simplicity
var GlobalService *Service

type Service struct {
	analytics

	DB *database.DB

	logger *log.Logger
	state  utils.State
}

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.start(); err != nil {
		s.logger.Error("Failed to start service: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *Service) start() error {
	if s.logger == nil {
		s.logger = logging.NewLogger(ServiceName)
	}

	now := time.Now()
	s.StartTime.Store(&now)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := s.stop(ctx); err != nil {
		s.logger.Error("Failed to stop service: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Service) stop(ctx context.Context) error {
	return nil
}

func (s *Service) State() (utils.StateCode, string) { return s.state.Get() }
