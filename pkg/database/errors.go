package database

import (
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
	return isError[*ValidationError](err)
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

func IsInternalError(err error) bool {
	return isError[*InternalError](err)
}

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

type FKElement interface {
	Table
	Identifier
	Namer
}

type ForeignKeyError struct {
	Msg   string
	Elems []FKElement
	Total uint64
}

func (err *ForeignKeyError) Error() string { return err.Msg }

type FkChecker[T FKElement] struct {
	db   ReadAccess
	cond condition
}

func CheckFKViolation[T FKElement](db ReadAccess, cond string, args ...any) *FkChecker[T] {
	return &FkChecker[T]{
		db:   db,
		cond: condition{sql: cond, args: args},
	}
}

func (fk *FkChecker[T]) Check(msg string) error {
	elem := utils.NewValue[T]()

	n, cErr := fk.db.Count(elem).Where(fk.cond.sql, fk.cond.args...).Run()
	if cErr != nil {
		return fmt.Errorf("failed to count %s elements: %w", elem.Appellation(), cErr)
	} else if n == 0 {
		return nil
	}

	const limit = 20

	var elems Slice[T]
	if err := fk.db.Select(&elems).Eager().Where(fk.cond.sql, fk.cond.args...).
		Limit(limit, 0).Run(); err != nil {
		return fmt.Errorf("failed to retrieve %s elements: %w", elem.Appellation(), err)
	}

	namedElems := make([]FKElement, len(elems))
	for i, elem := range elems {
		namedElems[i] = elem
	}

	return &ForeignKeyError{Msg: msg, Elems: namedElems, Total: n}
}

func (fk *FkChecker[T]) Checkf(msg string, args ...any) error {
	return fk.Check(fmt.Sprintf(msg, args...))
}
