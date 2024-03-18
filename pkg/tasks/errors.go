package tasks

import (
	"errors"
)

var (
	errBadTaskArguments = errors.New("bad arguments for tasks")
	errCommandTimeout   = errors.New("an external command timed out")
)

type warningError struct {
	msg string
}

func (e *warningError) Error() string {
	return e.msg
}
