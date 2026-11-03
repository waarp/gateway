package analytics

import (
	"context"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ServiceName = "Analytics"

//nolint:gochecknoglobals //global var is better for simplicity
var GlobalService *Service

type Service struct {
	analytics

	DB *database.DB

	logger       *log.Logger
	state        utils.State
	stopSampling context.CancelFunc
}

func (s *Service) Name() string { return ServiceName }

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return nil
	}

	if err := s.start(); err != nil {
		s.logger.Errorf("Failed to start service: %v", err)
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

	ctx, cancel := context.WithCancel(context.Background())
	s.stopSampling = cancel

	go s.sampleEvery(ctx, SamplePeriod)

	return nil
}

// sampleEvery samples the counters now, then every period, until ctx ends.
func (s *Service) sampleEvery(ctx context.Context, period time.Duration) {
	s.Sample(time.Now())

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.Sample(now)
		}
	}
}

func (s *Service) Stop(context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := s.stop(); err != nil {
		s.logger.Errorf("Failed to stop service: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Service) stop() error {
	if s.stopSampling != nil {
		s.stopSampling()
	}

	return nil
}

func (s *Service) State() (utils.StateCode, string) { return s.state.Get() }
