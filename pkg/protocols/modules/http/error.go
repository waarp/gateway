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

func readResponseMsg(resp *http.Response) string {
	msg := &strings.Builder{}
	//nolint:errcheck //we don't care about the error here
	if n, _ := io.CopyN(msg, resp.Body, httpErrMsgMaxSize); n > 0 {
		return msg.String()
	}

	return ""
}

func newUnexpectedStatusError(resp *http.Response) *UnexpectedStatusError {
	return &UnexpectedStatusError{
		method: resp.Request.Method,
		code:   resp.StatusCode,
		msg:    readResponseMsg(resp),
	}
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
