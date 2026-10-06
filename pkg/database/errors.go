package database

import (
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// ValidationError is the error returned when the entry given for insertion is
// not valid.
type ValidationError struct {
	err error
}

func (v *ValidationError) Error() string {
	return v.err.Error()
}

func (v *ValidationError) Unwrap() error { return errors.Unwrap(v.err) }

func IsValidationError(err error) bool {
	var ve *ValidationError

	return errors.As(err, &ve)
}

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
	return isError[*NotFoundError](err)
}

// NewNotFoundError returns a new validation `Error` for the given entry.
func NewNotFoundError(elem Table) *NotFoundError {
	return &NotFoundError{
		msg: fmt.Sprintf("%s not found", elem.Appellation()),
	}
}

// NewValidationError returns a new validation `Error` with the given message.
func NewValidationError(msg string) *ValidationError {
	//nolint:err113 //this is used to wrap errors
	return &ValidationError{err: errors.New(msg)}
}

// NewValidationErrorf returns a new validation `Error` with the given formatted message.
func NewValidationErrorf(msg string, args ...any) *ValidationError {
	//nolint:err113 //this is used to wrap errors
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

func isError[T error](err error) bool {
	_, ok := errors.AsType[T](err)

	return ok
}

const (
	// MySQL/MariaDB error numbers.
	mysqlErrLockWaitTimeout = 1205
	mysqlErrDeadlock        = 1213

	// PostgreSQL SQLSTATE codes.
	pgErrSerializationFailure = "40001"
	pgErrDeadlockDetected     = "40P01"
	pgErrLockNotAvailable     = "55P03"
)

func isRetryable(err error) bool {
	msErr, isMsErr := errors.AsType[*mysql.MySQLError](err)
	pgErr, isPgErr := errors.AsType[*pgconn.PgError](err)

	switch {
	case isMsErr:
		return msErr.Number == mysqlErrDeadlock || msErr.Number == mysqlErrLockWaitTimeout
	case isPgErr:
		return pgErr.Code == pgErrDeadlockDetected ||
			pgErr.Code == pgErrSerializationFailure ||
			pgErr.Code == pgErrLockNotAvailable
	default:
		return false
	}
}
