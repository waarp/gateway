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

func badRequest(msg string) *badRequestError {
	return &badRequestError{msg: msg}
}

func badRequestf(format string, args ...any) *badRequestError {
	return &badRequestError{msg: fmt.Sprintf(format, args...)}
}

func notFound(msg string) *notFoundError {
	return &notFoundError{msg: msg}
}

func notFoundf(format string, args ...any) *notFoundError {
	return &notFoundError{msg: fmt.Sprintf(format, args...)}
}

func internalf(format string, args ...any) *internalError {
	return &internalError{msg: fmt.Sprintf(format, args...)}
}

func isNotFound(err error) bool {
	var nf *notFoundError

	return errors.As(err, &nf)
}
