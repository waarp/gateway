package tasks

import (
	"fmt"
	"os"
)

var errWarning = fmt.Errorf("warning")

// Converts OS-specific file errors into normalized versions.
func normalizeFileError(err error) error {
	if os.IsNotExist(err) {
		e := err.(*os.PathError)
		return &ErrFileNotFound{e.Op, e.Path}
	}
	if os.IsPermission(err) {
		e := err.(*os.PathError)
		return &ErrFileNotFound{e.Op, e.Path}
	}
	return err
}

type ErrFileNotFound struct {
	action, path string
}

func FileNotFound(path string, op ...string) *ErrFileNotFound {
	if len(op) > 0 {
		return &ErrFileNotFound{op[0], path}
	}
	return &ErrFileNotFound{"open", path}
}

func (e *ErrFileNotFound) Error() string {
	return fmt.Sprintf("cannot open file '%s' - file does not exist", e.path)
}

type ErrPermissionDenied struct {
	path string
}

func (e *ErrPermissionDenied) Error() string {
	return fmt.Sprintf("cannot open file '%s' - permission denied", e.path)
}
