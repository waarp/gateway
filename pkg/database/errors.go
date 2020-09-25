package database

import (
	"fmt"
)

var (
	// ErrServiceUnavailable is the error returned by database operation
	// methods when the database is inactive
	ErrServiceUnavailable = &InternalError{
		msg:   "the database service is not running",
		cause: fmt.Errorf(""),
	}

	// ErrNilRecord is the error returned by database operation when the object
	// Which should be  used to generate the query or used to unmarshal the
	// query result is nil
	ErrNilRecord = &InputError{
		msg: "the record cannot be nil",
	}
)

// ValidationError is the error returned when the entry given for insertion is
// not valid.
type ValidationError struct {
	msg string
}

func (v *ValidationError) Error() string {
	return v.msg
}

// InputError is the error returned when the given bean does not correspond to
// a database table.
type InputError struct {
	msg string
}

func (i *InputError) Error() string {
	return i.msg
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
type NotFoundError struct {
	msg string
}

func (n *NotFoundError) Error() string {
	return n.msg
}

func newNotFoundError(elem string) *NotFoundError {
	return &NotFoundError{
		msg: fmt.Sprintf("%s not found", elem),
	}
}

// IsNotFound returns whether the given error is of type NotFoundError.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// NewValidationError returns a new validation `Error` with the given formatted message.
func NewValidationError(msg string, args ...interface{}) *ValidationError {
	return &ValidationError{
		msg: fmt.Sprintf(msg, args...),
	}
}

// NewInternalError returns a new internal `Error` with the given formatted message.
func NewInternalError(err error, msg string, args ...interface{}) *InternalError {
	return &InternalError{
		cause: err,
		msg:   fmt.Sprintf(msg, args...),
	}
}
