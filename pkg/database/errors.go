package database

import (
	"errors"
	"fmt"
)

var ErrUnsupportedReset = errors.New("unsupported operation 'ResetIncrement'")

// ValidationError is the error returned when the entry given for insertion is
// not valid.
type ValidationError struct {
	err error
}

func (v *ValidationError) Error() string {
	return v.err.Error()
}

func (v *ValidationError) Unwrap() error { return errors.Unwrap(v.err) }

// InternalError is the error encapsulating the database driver errors.
type InternalError struct {
	msg   string
	cause error
}

func (i *InternalError) Error() string {
	return fmt.Sprintf("%s: %v", i.msg, i.cause)
}

// Unwrap returns the wrapped error.
func (i *InternalError) Unwrap() error { return i.cause }

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
	//nolint:goerr113 //this is used to wrap errors
	return &ValidationError{err: fmt.Errorf(msg, args...)}
}

func WrapAsValidationError(err error) *ValidationError {
	return &ValidationError{err: err}
}

// NewInternalError returns a new internal `Error` with the given formatted message.
func NewInternalError(err error) *InternalError {
	return &InternalError{
		cause: err,
		msg:   "internal database error",
	}
}
