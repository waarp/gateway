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
	ContractRefreshServiceName = "EBICS Contract Refresh"
	defaultContractRefreshPoll = 30 * time.Second
)

type contractRefreshTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type contractRefreshTimeTicker struct {
	ticker *time.Ticker
}

func (t *contractRefreshTimeTicker) Chan() <-chan time.Time { return t.ticker.C }
func (t *contractRefreshTimeTicker) Stop()                  { t.ticker.Stop() }

type contractRefreshTickerFactory func(time.Duration) contractRefreshTicker

type contractRefreshRunner func(context.Context, *database.DB, int64, int64, bool) (*ContractRefreshResult, error)

// ContractRefreshService runs scheduled client-side EBICS contract refresh policies.
type ContractRefreshService struct {
	db     *database.DB
	logger *log.Logger
	state  utils.State

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.Mutex

	now       func() time.Time
	newTicker contractRefreshTickerFactory
	run       contractRefreshRunner
}

func NewContractRefreshService(db *database.DB) *ContractRefreshService {
	return &ContractRefreshService{
		db:    db,
		state: utils.NewState(utils.StateOffline, ""),
		now:   func() time.Time { return time.Now().UTC() },
		newTicker: func(d time.Duration) contractRefreshTicker {
			return &contractRefreshTimeTicker{ticker: time.NewTicker(d)}
		},
		run: RefreshContractViews,
	}
}

func (s *ContractRefreshService) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if s.logger == nil {
		s.logger = logging.NewLogger(ContractRefreshServiceName)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	ticker := s.newTicker(defaultContractRefreshPoll)

	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()

	s.wg.Add(1)
	go s.loop(runCtx, ticker)
	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *ContractRefreshService) Stop(context.Context) error {
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

func (s *ContractRefreshService) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *ContractRefreshService) loop(ctx context.Context, ticker contractRefreshTicker) {
	defer s.wg.Done()
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.Chan():
			if err := s.runDuePoliciesAt(ctx, s.now()); err != nil {
				s.state.Set(utils.StateError, err.Error())
				s.logger.Warningf("EBICS contract refresh scheduling failed: %v", err)

				continue
			}

			s.state.Set(utils.StateRunning, "")
		}
	}
}

func (s *ContractRefreshService) runDuePoliciesAt(ctx context.Context, now time.Time) error {
	now = now.UTC()

	var policies model.EbicsContractRefreshPolicies
	if err := s.db.Select(&policies).Owner().
		Where("enabled=? AND next_run_at<=?", true, now).
		OrderBy("next_run_at", true).
		Run(); err != nil {
		return fmt.Errorf("load due EBICS contract refresh policies: %w", err)
	}

	for _, policy := range policies {
		if err := s.runPolicy(ctx, now, policy); err != nil {
			return err
		}
	}

	return nil
}

func (s *ContractRefreshService) runPolicy(
	ctx context.Context,
	now time.Time,
	policy *model.EbicsContractRefreshPolicy,
) error {
	update := *policy
	update.Status = "RUNNING"
	update.LastAttemptAt = now
	if err := s.db.Update(&update).Run(); err != nil {
		return fmt.Errorf("mark EBICS contract refresh policy %q as running: %w", policy.Name, err)
	}

	_, runErr := s.run(ctx, s.db, policy.ClientID, policy.EbicsSubscriberID, policy.IncludeHEV)

	update = *policy
	update.LastAttemptAt = now
	update.NextRunAt = now.Add(policy.Interval())
	if runErr != nil {
		update.Status = "ERROR"
		update.LastError = runErr.Error()
		if err := s.db.Update(&update).Run(); err != nil {
			return fmt.Errorf("persist EBICS contract refresh policy %q failure: %w", policy.Name, err)
		}

		return nil
	}

	update.Status = "READY"
	update.LastSuccessAt = now
	update.LastError = ""
	if err := s.db.Update(&update).Run(); err != nil {
		return fmt.Errorf("persist EBICS contract refresh policy %q success: %w", policy.Name, err)
	}

	return nil
}
