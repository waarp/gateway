package amqp10

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errWriteOnDownloadRule = errors.New("illegal call to Write on an AMQP 1.0 server download rule")

type serverPipeline struct {
	pipeline *pipeline.Pipeline
	file     *pipeline.FileStream

	ctx    context.Context
	cancel context.CancelCauseFunc
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

func (s *serverPipeline) Write(p []byte) (int, error) {
	if s.pipeline.TransCtx.Rule.IsSend {
		return 0, errWriteOnDownloadRule
	}

	return utils.RWWithCtx(s.ctx, s.file.Write, p)
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

func (s *serverPipeline) Pause(context.Context) error {
	s.cancel(pipeline.NewError(types.TeStopped, "transfer paused by user"))
	return nil
}

func (s *serverPipeline) Interrupt(context.Context) error {
	s.cancel(pipeline.NewError(types.TeShuttingDown, "service is shutting down"))
	return nil
}

func (s *serverPipeline) Cancel(context.Context) error {
	s.cancel(pipeline.NewError(types.TeCanceled, "transfer canceled by user"))
	return nil
}
