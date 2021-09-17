package rest

import (
	"fmt"
)

type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

type forbidden struct{ msg string }

func (f *forbidden) Error() string { return f.msg }

type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }

func badRequest(format string, args ...interface{}) *badRequestError {
	return &badRequestError{msg: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...interface{}) *notFoundError {
	return &notFoundError{msg: fmt.Sprintf(format, args...)}
}
