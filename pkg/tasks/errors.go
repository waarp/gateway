package tasks

import (
	"errors"
	"fmt"
	"os"
)

var (
	errWarning          = fmt.Errorf("warning")
	errBadTaskArguments = errors.New("bad arguments for tasks")
	errCommandTimeout   = errors.New("an external command timed out")
)

// Converts OS-specific file errors into normalized versions.
func normalizeFileError(err error) error {
	var e *os.PathError

	ok := errors.As(err, &e)

	if os.IsNotExist(err) && ok {
		return &FileNotFoundError{e.Op, e.Path}
	}

	if os.IsPermission(err) && ok {
		return &PermissionDeniedError{e.Op, e.Path}
	}

	return err
}

type FileNotFoundError struct {
	action, path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("cannot %s file '%s' - file does not exist", e.action, e.path)
}

type PermissionDeniedError struct {
	action, path string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("cannot %s file '%s' - permission denied", e.action, e.path)
}
