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

// initPipeline initializes the pipeline.
func initPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	setTrace func() pipeline.Trace,
) (*serverPipeline, error) {
	var tErr error

	trans, tErr = pipeline.GetOldTransfer(db, logger, trans)
	if tErr != nil {
		return nil, toSFTPErr(tErr)
	}

	pip, tErr := pipeline.NewServerPipeline(db, trans)
	if tErr != nil {
		return nil, toSFTPErr(tErr)
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
func newServerPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	setTrace func() pipeline.Trace,
) (*serverPipeline, error) {
	servPip, err := initPipeline(db, logger, trans, setTrace)
	if err != nil {
		return nil, err
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

	s.pipeline.SetError(fromSFTPErr(err, types.TeConnectionReset, s.pipeline))
}

// Pause pauses the transfer and set the transfer's error to the pause error, so
// that it can be sent to the remote client.
func (s *serverPipeline) Pause(context.Context) error {
	sigPause := &types.TransferError{Code: types.TeStopped, Details: "transfer paused by user"}
	s.cancel(sigPause)

	return nil
}

// Interrupt interrupts the transfer and set the transfer's error to the shutdown
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Interrupt(context.Context) error {
	sigShutdown := &types.TransferError{Code: types.TeShuttingDown, Details: "service is shutting down"}
	s.cancel(sigShutdown)

	return nil
}

// Cancel cancels the transfer and set the transfer's error to the canceled
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Cancel(context.Context) error {
	sigCancel := &types.TransferError{Code: types.TeCanceled, Details: "transfer canceled by user"}
	s.cancel(sigCancel)

	return nil
}

// Execute pre-tasks & open transfer stream.
func (s *serverPipeline) init() error {
	return utils.RunWithCtx(s.ctx, func() error {
		if err := s.pipeline.PreTasks(); err != nil {
			return toSFTPErr(err)
		}

		var err error
		if s.file, err = s.pipeline.StartData(); err != nil {
			return toSFTPErr(err)
		}

		return nil
	})
}

// ReadAt reads the requested part of the transfer file.
func (s *serverPipeline) ReadAt(p []byte, off int64) (int, error) {
	var n int

	err := utils.RunWithCtx(s.ctx, func() error {
		var err error
		n, err = s.file.ReadAt(p, off)

		return toSFTPErr(err)
	})

	return n, err
}

// WriteAt writes the given data to the transfer file.
func (s *serverPipeline) WriteAt(p []byte, off int64) (int, error) {
	var n int

	err := utils.RunWithCtx(s.ctx, func() error {
		var err error
		n, err = s.file.WriteAt(p, off)

		return toSFTPErr(err)
	})

	return n, err
}

// Close file, executes post-tasks & end transfer.
func (s *serverPipeline) Close() error {
	return utils.RunWithCtx(s.ctx, func() error {
		if err := s.pipeline.EndData(); err != nil {
			return toSFTPErr(err)
		}

		if err := s.pipeline.PostTasks(); err != nil {
			return toSFTPErr(err)
		}

		if err := s.pipeline.EndTransfer(); err != nil {
			return toSFTPErr(err)
		}

		return nil
	})
}
