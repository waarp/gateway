package sftp

import (
	"context"
	"errors"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// serverPipeline is a struct used by SFTP servers to make SFTP transfers. It
// contains a pipeline, and implements the necessary functions to allow transfer
// interruption.
type serverPipeline struct {
	pipeline *pipeline.Pipeline
	file     *pipeline.FileStream

	ctx    context.Context
	cancel func(cause error)
}

func mkServerTransfer(db *database.DB, filepath string, account *model.LocalAccount,
	rule *model.Rule,
) (*model.Transfer, error) {
	if trans, err := pipeline.GetAvailableTransferByFilename(db, filepath, "",
		account, rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		return nil, toSFTPErr(err)
	}

	return pipeline.MakeServerTransfer("", filepath, account, rule), nil
}

// initPipeline initializes the pipeline.
func initPipeline(db *database.DB, logger *log.Logger, filepath string,
	account *model.LocalAccount, rule *model.Rule, size int64, setTrace func() pipeline.Trace,
) (*serverPipeline, error) {
	trans, tErr := mkServerTransfer(db, filepath, account, rule)
	if tErr != nil {
		return nil, tErr
	}

	if !rule.IsSend {
		trans.Filesize = size
	}

	pip, pErr := pipeline.NewServerPipeline(db, logger, trans, snmp.GlobalService)
	if pErr != nil {
		return nil, toSFTPErr(pErr)
	}

	if setTrace != nil {
		pip.Trace = setTrace()
	}

	ctx, cancel := context.WithCancelCause(context.Background())

	servPip := &serverPipeline{
		pipeline: pip,
		ctx:      ctx,
		cancel:   cancel,
	}

	pip.SetInterruptionHandlers(servPip.Pause, servPip.Interrupt, servPip.Cancel)

	return servPip, nil
}

// newServerPipeline creates a new serverPipeline, executes the transfer's
// pre-tasks, and returns the pipeline.
func newServerPipeline(db *database.DB, logger *log.Logger, filepath string,
	account *model.LocalAccount, rule *model.Rule, size int64, setTrace func() pipeline.Trace,
) (*serverPipeline, error) {
	servPip, pErr := initPipeline(db, logger, filepath, account, rule, size, setTrace)
	if pErr != nil {
		return nil, pErr
	}

	if err := servPip.init(); err != nil {
		return nil, err
	}

	return servPip, nil
}

var ErrSessionClosed = errors.New("session closed unexpectedly")

// TransferError is the function called when an error occurred on the remote
// client (or when the connection to the client is lost).
func (s *serverPipeline) TransferError(err error) {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		// "unexpected EOF" is not very meaningful as an error message, so we
		// change it to "session closed unexpectedly"
		err = ErrSessionClosed
	}

	s.pipeline.SetError(types.TeConnectionReset, err.Error())
}

// Pause pauses the transfer and set the transfer's error to the pause error, so
// that it can be sent to the remote client.
func (s *serverPipeline) Pause(context.Context) error {
	sigPause := pipeline.NewError(types.TeStopped, "transfer paused by user")
	s.cancel(sigPause)

	return nil
}

// Interrupt interrupts the transfer and set the transfer's error to the shutdown
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Interrupt(context.Context) error {
	sigShutdown := pipeline.NewError(types.TeShuttingDown, "service is shutting down")
	s.cancel(sigShutdown)

	return nil
}

// Cancel cancels the transfer and set the transfer's error to the canceled
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Cancel(context.Context) error {
	sigCancel := pipeline.NewError(types.TeCanceled, "transfer canceled by user")
	s.cancel(sigCancel)

	return nil
}

// Execute pre-tasks & open transfer stream.
func (s *serverPipeline) init() error {
	return utils.RunWithCtx(s.ctx, func() error {
		if err := s.pipeline.PreTasks(); err != nil {
			return pipeline.NewError(err.Code(), "pre-tasks failed")
		}

		var err *pipeline.Error
		if s.file, err = s.pipeline.StartData(); err != nil {
			return toSFTPErr(err)
		}

		return nil
	})
}

// ReadAt reads the requested part of the transfer file.
func (s *serverPipeline) ReadAt(p []byte, off int64) (int, error) {
	return utils.RWatWithCtx(s.ctx, s.file.ReadAt, p, off)
}

// WriteAt writes the given data to the transfer file.
func (s *serverPipeline) WriteAt(p []byte, off int64) (int, error) {
	return utils.RWatWithCtx(s.ctx, s.file.WriteAt, p, off)
}

// Close file, executes post-tasks & end transfer.
func (s *serverPipeline) Close() error {
	return utils.RunWithCtx(s.ctx, func() error {
		if err := s.pipeline.EndData(); err != nil {
			return pipeline.NewError(err.Code(), "failed to close file")
		}

		if err := s.pipeline.PostTasks(); err != nil {
			return pipeline.NewError(err.Code(), "post-tasks failed")
		}

		if err := s.pipeline.EndTransfer(); err != nil {
			return pipeline.NewError(err.Code(), "failed to finalize transfer")
		}

		return nil
	})
}
