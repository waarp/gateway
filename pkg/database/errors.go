package database

import (
	"errors"
	"fmt"
)

var ErrUnsupportedReset = errors.New("unsupported operation 'ResetIncrement'")

// ValidationError is the error returned when the entry given for insertion is
// not valid.
type ValidationError struct {
	msg string
}

func (v *ValidationError) Error() string {
	return v.msg
}

// InternalError is the error encapsulating the database driver errors.
type InternalError struct {
	msg   string
	cause error
}

func (i *InternalError) Error() string {
	return i.msg
}

// Unwrap returns the wrapped error.
func (i *InternalError) Unwrap() error {
	return i.cause
}

// NotFoundError is the error returned when the requested element in a 'Get',
// 'Update' or 'Delete' command could not be found.
type NotFoundError struct{ msg string }

func (n *NotFoundError) Error() string {
	return n.msg
}

// IsNotFound returns whether the given error is of type NotFoundError.
func IsNotFound(err error) bool {
	var nf *NotFoundError

	return errors.As(err, &nf)
}

// NewNotFoundError returns a new validation `Error` for the given entry.
func NewNotFoundError(elem Table) *NotFoundError {
	return &NotFoundError{
		msg: fmt.Sprintf("%s not found", elem.Appellation()),
	}
}

// NewValidationError returns a new validation `Error` with the given formatted message.
func NewValidationError(msg string, args ...interface{}) *ValidationError {
	return &ValidationError{
		msg: fmt.Sprintf(msg, args...),
	}
}

// NewInternalError returns a new internal `Error` with the given formatted message.
func NewInternalError(err error) *InternalError {
	return &InternalError{
		cause: err,
		msg:   "internal database error",
	}
}
