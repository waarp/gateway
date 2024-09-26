package rest

import (
	"errors"
	"fmt"
)

type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

type forbidden struct{ msg string }

func (f *forbidden) Error() string { return f.msg }

type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }

type internalError struct{ msg string }

func (i *internalError) Error() string { return i.msg }

func badRequest(format string, args ...interface{}) *badRequestError {
	return &badRequestError{msg: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...interface{}) *notFoundError {
	return &notFoundError{msg: fmt.Sprintf(format, args...)}
}

func internal(format string, args ...interface{}) *internalError {
	return &internalError{msg: fmt.Sprintf(format, args...)}
}

func isNotFound(err error) bool {
	var nf *notFoundError

	return errors.As(err, &nf)
}
