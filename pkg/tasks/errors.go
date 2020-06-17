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
		return &errFileNotFound{e.Op, e.Path}
	}
	if os.IsPermission(err) {
		e := err.(*os.PathError)
		return &errPermissionDenied{e.Op, e.Path}
	}
	return err
}

type errFileNotFound struct {
	action, path string
}

func (e *errFileNotFound) Error() string {
	return fmt.Sprintf("cannot %s file '%s' - file does not exist", e.action, e.path)
}

type errPermissionDenied struct {
	action, path string
}

func (e *errPermissionDenied) Error() string {
	return fmt.Sprintf("cannot %s file '%s' - permission denied", e.action, e.path)
}
