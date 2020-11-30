package r66

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
)

type stream struct {
	*pipeline.TransferStream
}

func (s *stream) ReadAt(b []byte, off int64) (int, error) {
	if s.Transfer.Step > types.StepData {
		return 0, io.EOF
	}
	n, err := s.TransferStream.ReadAt(b, off)
	if pErr, ok := err.(*model.PipelineError); ok {
		return n, pipelineToR66(pErr)
	}
	return n, err
}

func (s *stream) WriteAt(b []byte, off int64) (int, error) {
	if s.Transfer.Step > types.StepData {
		return len(b), nil
	}
	n, err := s.TransferStream.WriteAt(b, off)
	if pErr, ok := err.(*model.PipelineError); ok {
		return n, pipelineToR66(pErr)
	}
	return n, err
}

func pipelineToR66(err *model.PipelineError) *r66.Error {
	switch err.Kind {
	case model.KindDatabase:
		return &r66.Error{Code: r66.Internal, Detail: "internal database error"}
	case model.KindInterrupt:
		return &r66.Error{Code: r66.Shutdown, Detail: "the service is shutting down"}
	case model.KindPause:
		return &r66.Error{Code: r66.StoppedTransfer, Detail: "transfer was paused"}
	case model.KindCancel:
		return &r66.Error{Code: r66.CanceledTransfer, Detail: "transfer was cancelled"}
	case model.KindTransfer:
		return &r66.Error{Code: err.Cause.Code.R66Code(), Detail: err.Error()}
	default:
		return &r66.Error{Code: r66.Unknown, Detail: err.Error()}
	}
}
