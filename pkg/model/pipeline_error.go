package model

import . "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

// ErrorKind states the origin of a transfer error. The error  handling varies
// depending on the kind of the error.
type ErrorKind byte

// Regroups the different kind of pipeline errors.
const (
	KindTransfer ErrorKind = iota
	KindInterrupt
	KindDatabase
	KindPause
	KindCancel
)

// PipelineError is the type regrouping all types of errors which can occur
// during a transfer. This includes all types of TransferError, DatabaseError,
// and all transfer signals (shutdown, cancel, pause...).
type PipelineError struct {
	Kind  ErrorKind
	Cause TransferError
}

func (p *PipelineError) Error() string {
	switch p.Kind {
	case KindInterrupt:
		return "transfer interrupted"
	case KindDatabase:
		return "database error"
	case KindPause:
		return "transfer paused"
	case KindCancel:
		return "transfer cancelled"
	default:
		return p.Cause.Error()
	}
}

// NewPipelineError returns a new TransferError encapsulated inside a
// PipelineError. This function can only generate "normal" transfer errors.
func NewPipelineError(code TransferErrorCode, msg string) *PipelineError {
	return &PipelineError{
		Cause: TransferError{
			Code:    code,
			Details: msg,
		},
	}
}
