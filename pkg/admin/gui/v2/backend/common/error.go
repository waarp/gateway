package common

import (
	"errors"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type Error struct {
	Code    int
	Message string
	Cause   error
}

func NewError(code int, message string) *Error {
	return &Error{Message: message, Code: code}
}

func NewErrorWith(code int, message string, cause error) *Error {
	if database.IsValidationError(cause) {
		code = http.StatusBadRequest
	}

	realCause := errors.Unwrap(cause)
	if realCause == nil {
		realCause = cause
	}

	return &Error{Message: message, Code: code, Cause: realCause}
}

func (e *Error) Unwrap() error { return e.Cause }
func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) HttpMessage() string {
	if e.Code == http.StatusBadRequest && e.Cause != nil {
		return e.Cause.Error()
	}

	return e.Error()
}

func SendError(w http.ResponseWriter, logger *log.Logger, err error) bool {
	if err == nil {
		return false
	}

	var e *Error
	if !errors.As(err, &e) {
		logger.Errorf("unexpected error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return true
	}

	switch e.Code {
	case http.StatusBadRequest:
		logger.Warning(e.Error())
	default:
		logger.Error(e.Error())
	}

	http.Error(w, e.HttpMessage(), e.Code)

	return true
}

func IsNotFound(err error) bool {
	var e *Error

	return errors.As(err, &e) && e.Code == http.StatusNotFound
}
