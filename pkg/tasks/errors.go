package tasks

import (
	"fmt"
	"os"
)

type errWarning struct {
	msg string
}

func (e *errWarning) Error() string {
	return e.msg
}

// Converts OS-specific file errors into normalized versions.
func normalizeFileError(action string, err error) error {
	if os.IsNotExist(err) {
		e := err.(*os.PathError)
		return &errFileNotFound{action, e.Path}
	}
	if os.IsPermission(err) {
		e := err.(*os.PathError)
		return &errPermissionDenied{action, e.Path}
	}
	return err
}

type errFileNotFound struct {
	action, path string
}

func (e *errFileNotFound) Error() string {
	return fmt.Sprintf("failed to %s '%s' - file does not exist", e.action, e.path)
}

type errPermissionDenied struct {
	action, path string
}

func (e *errPermissionDenied) Error() string {
	return fmt.Sprintf("failed to %s '%s' - permission denied", e.action, e.path)
}
