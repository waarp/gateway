package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const httpErrMsgMaxSize = 1024

type UnexpectedStatusError struct {
	method string
	code   int
	msg    string
}

func newUnexpectedStatusError(resp *http.Response) *UnexpectedStatusError {
	statusError := &UnexpectedStatusError{
		method: resp.Request.Method,
		code:   resp.StatusCode,
	}

	msg := &strings.Builder{}
	//nolint:errcheck //we don't care about the error here
	if n, _ := io.CopyN(msg, resp.Body, httpErrMsgMaxSize); n > 0 {
		statusError.msg = msg.String()
	}

	return statusError
}

func (e *UnexpectedStatusError) Error() string {
	if e.msg == "" {
		return fmt.Sprintf("unexpected status code %d for %s", e.code, e.method)
	}

	return fmt.Sprintf("unexpected status code %d for %s: %s", e.code, e.method, e.msg)
}

func (e *UnexpectedStatusError) Method() string { return e.method }
func (e *UnexpectedStatusError) Code() int      { return e.code }
func (e *UnexpectedStatusError) Msg() string    { return e.msg }
