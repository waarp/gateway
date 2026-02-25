package webdav

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrReadOnUpload    = errors.New(`illegal call to "Read" on upload request`)
	ErrWriteOnDownload = errors.New(`illegal call to "Write" on download request`)
	ErrListOnFile      = errors.New(`illegal call to "Readdir" on file`)
)

type serverPipeline struct {
	pipeline *pipeline.Pipeline
	file     *pipeline.FileStream

	ctx    context.Context
	cancel func(cause error)
}

func (s *serverPipeline) init() error {
	if err := s.pipeline.PreTasks(); err != nil {
		return fmt.Errorf("pre-tasks failed: %w", err)
	}

	var err *pipeline.Error
	if s.file, err = s.pipeline.StartData(); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	return nil
}

func (s *serverPipeline) Stat() (fs.FileInfo, error) {
	//nolint:wrapcheck //no need to wrap here
	return s.file.Stat()
}

func (s *serverPipeline) Read(p []byte) (n int, err error) {
	if !s.pipeline.TransCtx.Rule.IsSend {
		return 0, ErrReadOnUpload
	}

	return utils.RWWithCtx(s.ctx, s.file.Read, p)
}

func (s *serverPipeline) Write(p []byte) (n int, err error) {
	if s.pipeline.TransCtx.Rule.IsSend {
		return 0, ErrWriteOnDownload
	}

	return utils.RWWithCtx(s.ctx, s.file.Write, p)
}

func (s *serverPipeline) Seek(offset int64, whence int) (int64, error) {
	//nolint:wrapcheck //no need to wrap here
	return s.file.Seek(offset, whence)
}

func (s *serverPipeline) Readdir(int) ([]fs.FileInfo, error) {
	return nil, ErrListOnFile
}

func (s *serverPipeline) Close() error {
	return utils.RunWithCtx(s.ctx, s.close)
}

func (s *serverPipeline) close() error {
	if err := s.pipeline.EndData(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	if err := s.pipeline.PostTasks(); err != nil {
		return fmt.Errorf("post-tasks failed: %w", err)
	}

	if err := s.pipeline.EndTransfer(); err != nil {
		return fmt.Errorf("failed to finalize transfer: %w", err)
	}

	return nil
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
