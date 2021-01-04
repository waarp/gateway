package database

import (
	"fmt"
)

// Error is the interface representing a database error. All database operations
// must return an error of this type to ensure (via type-checking) that database
// errors are correctly handled internally.
type Error interface {
	error
	db()
}

// ValidationError is the error returned when the entry given for insertion is
// not valid.
type ValidationError struct {
	msg string
}

func (v *ValidationError) Error() string {
	return v.msg
}

func (*ValidationError) db() {}

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

func (i *InternalError) db() {}

// NotFoundError is the error returned when the requested element in a 'Get',
// 'Update' or 'Delete' command could not be found.
type NotFoundError struct {
	msg string
}

func (n *NotFoundError) Error() string {
	return n.msg
}

func (*NotFoundError) db() {}

// IsNotFound returns whether the given error is of type NotFoundError.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
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
