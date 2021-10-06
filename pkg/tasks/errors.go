package tasks

import (
	"errors"
	"fmt"
	"os"
)

var (
	errBadTaskArguments = errors.New("bad arguments for tasks")
	errCommandTimeout   = errors.New("an external command timed out")
)

type errWarning struct {
	msg string
}

func (e *errWarning) Error() string {
	return e.msg
}

type fileNotFoundError struct {
	action, path string
}

func (e *fileNotFoundError) Error() string {
	return fmt.Sprintf("failed to %s '%s' - file does not exist", e.action, e.path)
}

type permissionDeniedError struct {
	action, path string
}

func (e *permissionDeniedError) Error() string {
	return fmt.Sprintf("failed to %s '%s' - permission denied", e.action, e.path)
}

// Converts OS-specific file errors into normalized versions.
func normalizeFileError(action string, err error) error {
	var e *os.PathError

	ok := errors.As(err, &e)

	if os.IsNotExist(err) && ok {
		return &fileNotFoundError{action, e.Path}
	}

	if os.IsPermission(err) && ok {
		return &permissionDeniedError{action, e.Path}
	}

	return err
}
