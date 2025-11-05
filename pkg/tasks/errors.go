package tasks

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var (
	ErrTransferInterrupted = newError(types.TeStopped, "transfer interrupted")
	ErrBadTaskArguments    = errors.New("bad arguments for tasks")
)

type Error struct {
	Code    types.TransferErrorCode
	Details string
	Cause   error
}

func newError(code types.TransferErrorCode, details string, args ...any) *Error {
	return &Error{Code: code, Details: fmt.Sprintf(details, args...)}
}

func newErrorWith(code types.TransferErrorCode, details string, cause error) *Error {
	return &Error{Code: code, Details: details, Cause: cause}
}

func (e *Error) Unwrap() error { return e.Cause }
func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Details
	}

	return fmt.Sprintf("%s: %v", e.Details, e.Cause)
}

type WarningError struct {
	msg string
}

func (e *WarningError) Error() string {
	return e.msg
}
