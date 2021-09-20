package r66

import (
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type stream struct {
	*pipeline.TransferStream
}

func (s *stream) ReadAt(b []byte, off int64) (int, error) {
	if s.Transfer.Step > types.StepData {
		return 0, io.EOF
	}

	n, err := s.TransferStream.ReadAt(b, off)

	var pErr *types.TransferError
	if ok := errors.As(err, &pErr); ok {
		return n, toR66Error(pErr)
	}

	if err != nil {
		return n, fmt.Errorf("cannot read stream: %w", err)
	}

	return n, nil
}

func (s *stream) WriteAt(b []byte, off int64) (int, error) {
	if s.Transfer.Step > types.StepData {
		return len(b), nil
	}

	n, err := s.TransferStream.WriteAt(b, off)

	var pErr *types.TransferError
	if ok := errors.As(err, &pErr); ok {
		return n, toR66Error(pErr)
	}

	if err != nil {
		return n, fmt.Errorf("cannot write stream: %w", err)
	}

	return n, nil
}

func toR66Error(err error) *r66.Error {
	var tErr types.TransferError
	if ok := errors.As(err, &tErr); ok {
		return &r66.Error{Code: tErr.Code.R66Code(), Detail: tErr.Details}
	}

	return &r66.Error{Code: r66.Unknown, Detail: err.Error()}
}
