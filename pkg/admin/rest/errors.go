package rest

import "fmt"

type errBadRequest struct{ msg string }

func (e *errBadRequest) Error() string { return e.msg }

type forbidden struct{ msg string }

func (f *forbidden) Error() string { return f.msg }

type errNotFound struct{ msg string }

func (e *errNotFound) Error() string { return e.msg }

func badRequest(format string, args ...interface{}) *errBadRequest {
	return &errBadRequest{msg: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...interface{}) *errNotFound {
	return &errNotFound{msg: fmt.Sprintf(format, args...)}
}
