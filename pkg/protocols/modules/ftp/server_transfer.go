package ftp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrRESTOnNewTransfer   = errors.New(`"REST" command is forbidden on new transfers`)
	ErrSendWithReceiveRule = errors.New("action forbidden: cannot send file with a receive rule")
	ErrStoreWithSendRule   = errors.New("action forbidden: cannot store file with a send rule")
)

type serverTransfer struct {
	pip    *pipeline.Pipeline
	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (s *serverFS) newServerTransfer(path string, isSend bool, offset int64,
) (*serverTransfer, error) {
	analytics.AddConnection()

	st, err := s.mkNewServerTransfer(path, isSend, offset)
	if err != nil {
		analytics.SubConnection()

		return nil, err
	}

	return st, nil
}

func (s *serverFS) mkNewServerTransfer(path string, isSend bool, offset int64,
) (*serverTransfer, error) {
	rule, ruleErr := protoutils.GetClosestRule(s.db, s.logger, s.dbServer, s.dbAcc, path, isSend)
	if ruleErr != nil {
		return nil, ruleErr //nolint:wrapcheck //no need to wrap here
	} else if rule.IsSend != isSend {
		if isSend {
			return nil, ErrSendWithReceiveRule
		}

		return nil, ErrStoreWithSendRule
	}

	realPath := strings.TrimLeft(path, "/")
	realPath = strings.TrimPrefix(realPath, rule.Path)
	realPath = strings.TrimLeft(realPath, "/")

	var trans *model.Transfer

	if offset != 0 { // Check for existing transfer
		var err error
		if trans, err = s.checkExistingTransfer(realPath, rule, offset); err != nil {
			return nil, err
		}

		trans.Progress = offset
	} else { // Create Transfer
		trans = &model.Transfer{
			RuleID:         rule.ID,
			LocalAccountID: utils.NewNullInt64(s.dbAcc.ID),
			Filesize:       model.UnknownSize,
			Start:          time.Now(),
			Status:         types.StatusRunning,
			Step:           types.StepNone,
		}

		if rule.IsSend {
			trans.SrcFilename = realPath
		} else {
			trans.DestFilename = realPath
		}
	}

	pip, pipErr := pipeline.NewServerPipeline(s.db, s.logger, trans, snmp.GlobalService)
	if pipErr != nil {
		return nil, pipErr
	}

	if s.tracer != nil {
		pip.Trace = s.tracer()
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	file := &serverTransfer{pip: pip, ctx: ctx, cancel: cancel}
	pip.SetInterruptionHandlers(file.Pause, file.Interrupt, file.Cancel)

	if err := file.init(); err != nil {
		return nil, err
	}

	return file, nil
}

func (s *serverFS) checkExistingTransfer(path string, rule *model.Rule, offset int64,
) (*model.Transfer, error) {
	var trans model.Transfer

	query := s.db.Get(&trans, "local_account_id = ? AND rule_id = ? AND status <> ?",
		s.dbAcc.ID, rule.ID, types.StatusRunning)
	if rule.IsSend {
		query = query.And("src_filename = ? AND progress >= ?", path, offset)
	} else {
		query = query.And("dest_filename = ? AND progress >= ?", path, offset)
	}

	if err := query.Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, ErrRESTOnNewTransfer
		}

		return nil, fmt.Errorf("failed to check for existing transfer: %w", err)
	}

	return &trans, nil
}

func (s *serverTransfer) init() error {
	if err := s.pip.PreTasks(); err != nil {
		return err
	}

	if _, err := s.pip.StartData(); err != nil {
		return err
	}

	return nil
}

func (s *serverTransfer) Name() string {
	if s.pip.Stream.TransCtx.Rule.IsSend {
		return s.pip.Stream.TransCtx.Transfer.SrcFilename
	} else {
		return s.pip.Stream.TransCtx.Transfer.DestFilename
	}
}

func (s *serverTransfer) Read(p []byte) (n int, err error) {
	return utils.RWWithCtx(s.ctx, s.pip.Stream.Read, p)
}

func (s *serverTransfer) ReadAt(p []byte, off int64) (n int, err error) {
	return utils.RWatWithCtx(s.ctx, s.pip.Stream.ReadAt, p, off)
}

func (s *serverTransfer) Write(p []byte) (n int, err error) {
	return utils.RWWithCtx(s.ctx, s.pip.Stream.Write, p)
}

func (s *serverTransfer) WriteAt(p []byte, off int64) (n int, err error) {
	return utils.RWatWithCtx(s.ctx, s.pip.Stream.WriteAt, p, off)
}

func (s *serverTransfer) Seek(offset int64, whence int) (int64, error) {
	//nolint:wrapcheck //no need to wrap here
	return s.pip.Stream.Seek(offset, whence)
}

func (s *serverTransfer) WriteString(str string) (int, error) {
	return s.Write([]byte(str))
}

func (s *serverTransfer) Close() error {
	defer analytics.SubConnection()

	isSend := s.pip.Stream.TransCtx.Rule.IsSend
	progress := s.pip.Stream.TransCtx.Transfer.Progress
	filesize := s.pip.Stream.TransCtx.Transfer.Filesize

	if isSend && progress < filesize {
		s.pip.SetError(types.TeConnectionReset, "data connection closed unexpectedly")

		return nil
	}

	return utils.RunWithCtx(s.ctx, func() error {
		if err := s.pip.EndData(); err != nil {
			return err
		}

		if err := s.pip.PostTasks(); err != nil {
			return err
		}

		if err := s.pip.EndTransfer(); err != nil {
			return err
		}

		return nil
	})
}

func (s *serverTransfer) TransferError(err error) {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		s.pip.SetError(types.TeConnectionReset, "data connection closed unexpectedly")

		return
	}

	s.pip.SetError(types.TeConnectionReset, err.Error())
}

func (s *serverTransfer) Pause(context.Context) error {
	sig := pipeline.NewError(types.TeStopped, "transfer paused by user")
	s.cancel(sig)

	return nil
}

func (s *serverTransfer) Interrupt(context.Context) error {
	sig := pipeline.NewError(types.TeShuttingDown, "transfer interrupted by service shutdown")
	s.cancel(sig)

	return nil
}

func (s *serverTransfer) Cancel(context.Context) error {
	sig := pipeline.NewError(types.TeCanceled, "transfer canceled by user")
	s.cancel(sig)

	return nil
}

//nolint:wrapcheck //no need to wrap here
func (s *serverTransfer) Stat() (os.FileInfo, error) { return s.pip.Stream.Stat() }

//nolint:wrapcheck //no need to wrap here
func (s *serverTransfer) Sync() error { return s.pip.Stream.Sync() }

func (s *serverTransfer) Readdirnames(int) ([]string, error) { return nil, fs.ErrNotDir }
func (s *serverTransfer) Readdir(int) ([]os.FileInfo, error) { return nil, fs.ErrNotDir }
func (s *serverTransfer) Truncate(int64) error               { return errNotImplemented }
