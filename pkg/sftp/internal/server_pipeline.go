package internal

import (
	"context"
	"errors"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

var (
	sigShutdown = &types.TransferError{Code: types.TeShuttingDown, Details: "service is shutting down"}
	sigPause    = &types.TransferError{Code: types.TeStopped, Details: "transfer paused by user"}
	sigCancel   = &types.TransferError{Code: types.TeCanceled, Details: "transfer cancelled by user"}
)

// ServerPipeline is a struct used by SFTP servers to make SFTP transfers. It
// contains a pipeline, and implements the necessary functions to allow transfer
// interruption.
type ServerPipeline struct {
	pipeline *pipeline.Pipeline
	file     pipeline.TransferStream

	transList  *service.TransferMap
	storage    *utils.ErrorStorage
	endSession func()
}

// initialises the pipeline
func initPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	endSession func(), transList *service.TransferMap) (*ServerPipeline, error) {

	var tErr *types.TransferError
	trans, tErr = pipeline.GetServerTransfer(db, logger, trans)
	if tErr != nil {
		return nil, ToSFTPErr(tErr)
	}
	pip, tErr := pipeline.NewServerPipeline(db, trans)
	if tErr != nil {
		return nil, ToSFTPErr(tErr)
	}

	servPip := &ServerPipeline{
		pipeline:   pip,
		transList:  transList,
		storage:    utils.NewErrorStorage(),
		endSession: endSession,
	}
	transList.Add(trans.ID, servPip)
	return servPip, nil
}

// NewServerPipeline creates a new ServerPipeline, executes the transfer's
// pre-tasks, and returns the pipeline.
func NewServerPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,
	transList *service.TransferMap, endSession func()) (*ServerPipeline, error) {

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
func (s *ServerPipeline) TransferError(err error) {
	s.pipeline.SetError(types.NewTransferError(types.TeConnectionReset,
		"session closed unexpectedly"))
	_ = s.handleError(err)
}

// Pause pauses the transfer and set the transfer's error to the pause error, so
// that it can be sent to the remote client.
func (s *ServerPipeline) Pause(ctx context.Context) error {
	s.pipeline.Pause(func() {
		s.storage.StoreCtx(ctx, sigPause)
	})
	s.endSession()
	return ctx.Err()
}

// Interrupt interrupts the transfer and set the transfer's error to the shutdown
// error, so that it can be sent to the remote client.
func (s *ServerPipeline) Interrupt(ctx context.Context) error {
	s.pipeline.Interrupt(func() {
		s.storage.StoreCtx(ctx, sigShutdown)
	})
	s.endSession()
	return ctx.Err()
}

// Cancel cancels the transfer and set the transfer's error to the cancelled
// error, so that it can be sent to the remote client.
func (s *ServerPipeline) Cancel(ctx context.Context) error {
	s.pipeline.Cancel(func() {
		s.storage.StoreCtx(ctx, sigCancel)
	})
	s.endSession()
	return ctx.Err()
}

func (s *ServerPipeline) handleError(err error) error {
	s.transList.Delete(s.pipeline.TransCtx.Transfer.ID)
	sErr := ToSFTPErr(err)
	s.storage.Store(sErr)
	return sErr
}

func (s *ServerPipeline) exec(f func() *types.TransferError) error {
	select {
	case <-s.storage.Wait():
		return errors.New("file handle is no longer valid")
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
		return s.storage.Get()
	case <-done:
		if err != nil {
			return s.handleError(err)
		}
		return nil
	}
}

// Execute pre-tasks & open transfer stream
func (s *ServerPipeline) init() error {
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
func (s *ServerPipeline) ReadAt(p []byte, off int64) (int, error) {
	select {
	case iErr, ok := <-s.storage.Wait():
		if ok {
			return 0, iErr
		}
		return 0, errors.New("file handle is no longer valid")
	default:
	}

	n, err := s.file.ReadAt(p, off)

	select {
	case <-s.storage.Wait():
		return n, s.storage.Get()
	default:
		if err != nil && err != io.EOF {
			return n, s.handleError(err)
		}
		return n, err
	}
}

// WriteAt writes the given data to the transfer file.
func (s *ServerPipeline) WriteAt(p []byte, off int64) (int, error) {
	select {
	case iErr, ok := <-s.storage.Wait():
		if ok {
			return 0, iErr
		}
		return 0, errors.New("file handle is no longer valid")
	default:
	}

	n, err := s.file.WriteAt(p, off)

	select {
	case <-s.storage.Wait():
		return n, s.storage.Get()
	default:
		if err != nil {
			return n, s.handleError(err)
		}
		return n, nil
	}
}

// Close file, executes post-tasks & end transfer
func (s *ServerPipeline) Close() error {
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
	return err
}
