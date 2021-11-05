package sftp

import (
	"context"
	"errors"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// serverPipeline is a struct used by SFTP servers to make SFTP transfers. It
// contains a pipeline, and implements the necessary functions to allow transfer
// interruption.
type serverPipeline struct {
	pipeline *pipeline.Pipeline
	file     pipeline.TransferStream

	transList  *service.TransferMap
	storage    *utils.ErrorStorage
	endSession func(context.Context)
}

// initPipeline initializes the pipeline.
func initPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	endSession func(context.Context), transList *service.TransferMap,
) (*serverPipeline, error) {
	var tErr *types.TransferError

	trans, tErr = pipeline.GetOldTransfer(db, logger, trans)
	if tErr != nil {
		return nil, toSFTPErr(tErr)
	}

	pip, tErr := pipeline.NewServerPipeline(db, trans)
	if tErr != nil {
		return nil, toSFTPErr(tErr)
	}

	servPip := &serverPipeline{
		pipeline:   pip,
		transList:  transList,
		storage:    utils.NewErrorStorage(),
		endSession: endSession,
	}

	transList.Add(trans.ID, servPip)

	return servPip, nil
}

// newServerPipeline creates a new serverPipeline, executes the transfer's
// pre-tasks, and returns the pipeline.
func newServerPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	transList *service.TransferMap, endSession func(context.Context),
) (*serverPipeline, error) {
	servPip, err := initPipeline(db, logger, trans, endSession, transList)
	if err != nil {
		return nil, err
	}

	if err := servPip.init(); err != nil {
		return nil, err
	}

	return servPip, nil
}

// TransferError is the function called when an error occurred on the remote
// client (or when the connection to the client is lost).
func (s *serverPipeline) TransferError(err error) {
	s.pipeline.SetError(types.NewTransferError(types.TeConnectionReset,
		"session closed unexpectedly"))
	//nolint:errcheck // the returned error is unimportant
	_ = s.handleError(err)
}

// Pause pauses the transfer and set the transfer's error to the pause error, so
// that it can be sent to the remote client.
func (s *serverPipeline) Pause(ctx context.Context) error {
	sigPause := &types.TransferError{Code: types.TeStopped, Details: "transfer paused by user"}

	s.pipeline.Pause(func() {
		s.storage.StoreCtx(ctx, sigPause)
	})
	s.endSession(ctx)

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

// Interrupt interrupts the transfer and set the transfer's error to the shutdown
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Interrupt(ctx context.Context) error {
	sigShutdown := &types.TransferError{Code: types.TeShuttingDown, Details: "service is shutting down"}

	s.pipeline.Interrupt(func() {
		s.storage.StoreCtx(ctx, sigShutdown)
	})
	s.endSession(ctx)

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

// Cancel cancels the transfer and set the transfer's error to the canceled
// error, so that it can be sent to the remote client.
func (s *serverPipeline) Cancel(ctx context.Context) error {
	sigCancel := &types.TransferError{Code: types.TeCanceled, Details: "transfer canceled by user"}

	s.pipeline.Cancel(func() {
		s.storage.StoreCtx(ctx, sigCancel)
	})
	s.endSession(ctx)

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (s *serverPipeline) handleError(err error) error {
	s.transList.Delete(s.pipeline.TransCtx.Transfer.ID)

	sErr := toSFTPErr(err)
	s.storage.Store(sErr)

	return sErr
}

func (s *serverPipeline) exec(f func() *types.TransferError) error {
	select {
	case <-s.storage.Wait():
		return errInvalidHandle
	default:
	}

	var err *types.TransferError

	done := make(chan struct{})

	go func() {
		err = f()

		close(done)
	}()

	select {
	case <-s.storage.Wait():
		return s.handleError(s.storage.Get())
	case <-done:
		if err != nil {
			return s.handleError(err)
		}

		return nil
	}
}

// Execute pre-tasks & open transfer stream.
func (s *serverPipeline) init() error {
	return s.exec(func() *types.TransferError {
		if tErr := s.pipeline.PreTasks(); tErr != nil {
			return tErr
		}

		file, tErr := s.pipeline.StartData()
		if tErr != nil {
			return tErr
		}
		s.file = file

		return nil
	})
}

// ReadAt reads the requested part of the transfer file.
func (s *serverPipeline) ReadAt(p []byte, off int64) (int, error) {
	select {
	case <-s.storage.Wait():
		return 0, toSFTPErr(s.storage.Get())
	default:
	}

	n, err := s.file.ReadAt(p, off)

	select {
	case <-s.storage.Wait():
		return n, s.handleError(s.storage.Get())
	default:
		if err != nil && !errors.Is(err, io.EOF) {
			return n, s.handleError(err)
		}

		return n, io.EOF
	}
}

// WriteAt writes the given data to the transfer file.
func (s *serverPipeline) WriteAt(p []byte, off int64) (int, error) {
	select {
	case <-s.storage.Wait():
		return 0, toSFTPErr(s.storage.Get())
	default:
	}

	n, err := s.file.WriteAt(p, off)

	select {
	case <-s.storage.Wait():
		return n, s.handleError(s.storage.Get())

	default:
		if err != nil {
			return n, s.handleError(err)
		}

		return n, nil
	}
}

// Close file, executes post-tasks & end transfer.
func (s *serverPipeline) Close() error {
	err := s.exec(func() *types.TransferError {
		if tErr := s.pipeline.EndData(); tErr != nil {
			return tErr
		}

		if tErr := s.pipeline.PostTasks(); tErr != nil {
			return tErr
		}

		return s.pipeline.EndTransfer()
	})

	s.storage.Close()
	s.transList.Delete(s.pipeline.TransCtx.Transfer.ID)

	return toSFTPErr(err)
}
