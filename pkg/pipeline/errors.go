package pipeline

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var (
	errPause       = NewError(types.TeStopped, "transfer paused by user")
	errInterrupted = NewError(types.TeShuttingDown, "transfer interrupted by a service shutdown")
	errCanceled    = NewError(types.TeCanceled, "transfer canceled by user")
)

// fileErrToTransferErr takes an error returned by a file operation function
// (like os.Open or os.Create) and returns the corresponding types.TransferError.
func fileErrToTransferErr(err error) *Error {
	if errors.Is(err, fs.ErrNotExist) {
		return NewError(types.TeFileNotFound, "file not found")
	}

	if errors.Is(err, fs.ErrPermission) {
		return NewError(types.TeForbidden, "file operation not allowed")
	}

	return NewErrorWith(types.TeUnknown, "file operation failed", err)
}

type Error struct {
	code    types.TransferErrorCode
	details string
	cause   error
}

func NewError(code types.TransferErrorCode, details string, args ...any) *Error {
	return &Error{code: code, details: fmt.Sprintf(details, args...)}
}

func NewErrorWith(code types.TransferErrorCode, details string, cause error) *Error {
	return &Error{code: code, details: details, cause: cause}
}

func (e *Error) Code() types.TransferErrorCode { return e.code }
func (e *Error) Unwrap() error                 { return e.cause }
func (e *Error) Redacted() string              { return e.details }

func (e *Error) Error() string {
	return fmt.Sprintf("TransferError(%s): %s", e.code, e.Redacted())
}

func (e *Error) Details() string {
	if e.cause == nil {
		return e.details
	}

	return fmt.Sprintf("%s: %v", e.details, e.cause)
}
