package osfs

import (
	"errors"
	"io/fs"
)

func pathError(op, path string, err error) error {
	if perr, ok := errors.AsType[*fs.PathError](err); ok {
		return perr
	}

	return &fs.PathError{Op: op, Path: path, Err: err}
}
