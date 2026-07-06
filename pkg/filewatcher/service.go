package filewatcher

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //global var is needed here
var Filewatchers = services.NewServiceMap[*Service]()

var ErrRuleSend = errors.New("cannot retrieve remote files with a send rule")

type Service struct {
	logger *log.Logger
	db     *database.DB
	fw     *model.FileWatcher

	state  utils.State
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewFilewatcher(db *database.DB, fw *model.FileWatcher) *Service {
	return &Service{
		db:     db,
		fw:     fw,
		logger: logging.NewLogger(fw.Flow),
	}
}

func (s *Service) Name() string                     { return s.fw.Flow }
func (s *Service) State() (utils.StateCode, string) { return s.state.Get() }

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.start(); err != nil {
		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := utils.RunWithCtx(ctx, func() error {
		s.cancel()
		s.wg.Wait()

		return nil
	}); err != nil {
		msg := fmt.Sprintf("Failed to stop filewatcher: %v", err)
		s.logger.Error(msg)
		s.state.Set(utils.StateError, msg)

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Service) start() error {
	if err := s.db.Get(s.fw, "id=?", s.fw.ID).Run(); err != nil {
		msg := fmt.Sprintf("Failed to retrieve filewatcher: %v", err)
		s.logger.Error(msg)
		s.state.Set(utils.StateError, msg)

		return fmt.Errorf("failed to retrieve filewatcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	if s.fw.Interval == 0 {
		return nil
	}

	ticker := time.NewTicker(s.fw.Interval)

	s.wg.Go(func() {
		defer cancel()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if err := s.processTick(s.fw); err != nil {
				s.logger.Errorf("Failed to process tick: %v", err)
			}
		}
	})

	return nil
}

func (s *Service) FireOnce(fw *model.FileWatcher) error {
	if err := s.processTick(fw); err != nil {
		s.logger.Errorf("Failed to process tick: %v", err)

		return err
	}

	return nil
}

func (s *Service) getTransferCtx(fw *model.FileWatcher) (*model.TransferContext, error) {
	fakeTransfer := &model.Transfer{
		RuleID:          fw.Rule.ID,
		RemoteAccountID: fw.RemoteAccount.NullableID(),
		ClientID:        fw.Client.NullableID(),
	}

	ctx, err := model.GetTransferContext(s.db, s.logger, fakeTransfer)
	if err != nil {
		return nil, fmt.Errorf("failed to make transfer context: %w", err)
	}

	if ctx.Rule.IsSend {
		return nil, ErrRuleSend
	}

	return ctx, nil
}

func (s *Service) processTick(fw *model.FileWatcher) error {
	ctx, err := s.getTransferCtx(fw)
	if err != nil {
		return err
	}

	lister, err := s.getLister(ctx)
	if err != nil {
		return err
	}

	files, listErr := lister.List(fw.Pattern)
	if listErr != nil {
		s.logger.Errorf("Failed to list files: %v", listErr)

		return fmt.Errorf("failed to list files: %w", listErr)
	}

	if !fw.NoDuplicateCheck {
		if files, err = s.removeDuplicates(files); err != nil {
			return err
		}
	}

	var transErrs []error

	for _, file := range files {
		trans := &model.Transfer{
			Status:          types.StatusPlanned,
			RuleID:          ctx.Rule.ID,
			RemoteAccountID: ctx.RemoteAccount.NullableID(),
			ClientID:        ctx.Client.NullableID(),
			SrcFilename:     file.Name(),
			Filesize:        file.Size(),
		}

		if dbErr := s.db.Insert(trans).Run(); dbErr != nil {
			s.logger.Errorf("Failed to insert transfer for file %q: %v", file, dbErr)

			transErrs = append(transErrs,
				fmt.Errorf("failed to insert transfer for file %q: %w", file, dbErr))
		}
	}

	return errors.Join(transErrs...)
}

func (s *Service) removeDuplicates(results []fs.FileInfo) ([]fs.FileInfo, error) {
	filtered := make([]fs.FileInfo, 0, len(results))

	for _, result := range results {
		var trans model.NormalizedTransferView
		if err := s.db.
			Get(&trans, "src_filename=?", result.Name()).
			And("filesize=?", result.Size()).
			And("start>?", result.ModTime().UTC()).
			Run(); database.IsNotFound(err) {
			// no duplicate transfer found, add to results
			filtered = append(filtered, result)
		} else if err != nil {
			return nil, fmt.Errorf("failed to retrieve transfer: %w", err)
		}
	}

	return filtered, nil
}
