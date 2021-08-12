package r66

import (
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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
	if pErr, ok := err.(*types.TransferError); ok {
		return n, toR66Error(pErr)
	}
	return n, err
}

func (s *stream) WriteAt(b []byte, off int64) (int, error) {
	if s.Transfer.Step > types.StepData {
		return len(b), nil
	}
	n, err := s.TransferStream.WriteAt(b, off)
	if pErr, ok := err.(*types.TransferError); ok {
		return n, toR66Error(pErr)
	}
	return n, err
}

func toR66Error(err error) *r66.Error {
	if tErr, ok := err.(types.TransferError); ok {
		return &r66.Error{Code: tErr.Code.R66Code(), Detail: tErr.Details}
	}
	return &r66.Error{Code: r66.Unknown, Detail: err.Error()}
}
