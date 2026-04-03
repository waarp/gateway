package ebics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	MaintenanceServiceName = "EBICS Maintenance"
)

type maintenanceTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type timeTicker struct {
	ticker *time.Ticker
}

func (t *timeTicker) Chan() <-chan time.Time { return t.ticker.C }
func (t *timeTicker) Stop()                  { t.ticker.Stop() }

type maintenanceTickerFactory func(time.Duration) maintenanceTicker

// MaintenanceService runs periodic EBICS maintenance tasks.
type MaintenanceService struct {
	db     *database.DB
	logger *log.Logger
	state  utils.State

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.Mutex

	interval  time.Duration
	now       func() time.Time
	newTicker maintenanceTickerFactory
}

// NewMaintenanceService creates the EBICS maintenance service.
func NewMaintenanceService(db *database.DB) *MaintenanceService {
	return &MaintenanceService{
		db:       db,
		state:    utils.NewState(utils.StateOffline, ""),
		interval: time.Duration(model.DefaultEbicsMaintenanceIntervalSeconds) * time.Second,
		now:      func() time.Time { return time.Now().UTC() },
		newTicker: func(d time.Duration) maintenanceTicker {
			return &timeTicker{ticker: time.NewTicker(d)}
		},
	}
}

// Start starts periodic EBICS maintenance.
func (s *MaintenanceService) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if s.logger == nil {
		s.logger = logging.NewLogger(MaintenanceServiceName)
	}

	policy, err := model.EnsureDefaultEbicsRuntimePolicy(s.db)
	if err != nil {
		return fmt.Errorf("ensure the default EBICS runtime policy: %w", err)
	}

	s.interval = policy.MaintenanceInterval()

	runCtx, cancel := context.WithCancel(context.Background())
	ticker := s.newTicker(s.interval)

	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()

	s.wg.Add(1)
	go s.loop(runCtx, ticker)
	s.state.Set(utils.StateRunning, "")

	return nil
}

// Stop stops periodic EBICS maintenance.
func (s *MaintenanceService) Stop(context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.mutex.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	s.wg.Wait()
	s.state.Set(utils.StateOffline, "")

	return nil
}

// State returns the current service state.
func (s *MaintenanceService) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *MaintenanceService) loop(ctx context.Context, ticker maintenanceTicker) {
	defer s.wg.Done()

	currentTicker := ticker
	defer currentTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-currentTicker.Chan():
			policy, err := model.EnsureDefaultEbicsRuntimePolicy(s.db)
			if err != nil {
				s.state.Set(utils.StateError, err.Error())
				s.logger.Warningf("EBICS maintenance failed to load policy: %v", err)

				continue
			}

			if nextInterval := policy.MaintenanceInterval(); nextInterval != s.interval {
				s.interval = nextInterval
				currentTicker.Stop()
				currentTicker = s.newTicker(s.interval)
			}

			if !policy.Enabled {
				s.state.Set(utils.StateRunning, "")

				continue
			}

			runErr := s.runMaintenanceAt(s.now(), policy)
			if runErr != nil {
				s.state.Set(utils.StateError, runErr.Error())
				s.logger.Warningf("EBICS maintenance failed: %v", runErr)

				continue
			}

			s.state.Set(utils.StateRunning, "")
		}
	}
}

func (s *MaintenanceService) runMaintenanceAt(now time.Time, policy *model.EbicsRuntimePolicy) error {
	now = now.UTC()

	if err := model.PurgeEbicsNoncesBefore(s.db, now); err != nil {
		return fmt.Errorf("purge expired EBICS nonces: %w", err)
	}

	if err := model.PurgeEbicsTransactionsBefore(s.db, now.Add(-policy.TransactionRetention())); err != nil {
		return fmt.Errorf("purge terminal EBICS transactions: %w", err)
	}

	if err := model.PurgeEbicsRTNEventsBefore(s.db, now.Add(-policy.RTNEventRetention())); err != nil {
		return fmt.Errorf("purge terminal EBICS RTN events: %w", err)
	}

	return nil
}
