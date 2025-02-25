package fs

import (
	"errors"
	gofs "io/fs"
	"os"
	"syscall"

	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/vfs"
)

var (
	ErrNotExist   = vfs.ENOENT
	ErrExist      = vfs.EEXIST
	ErrPermission = vfs.EPERM
	ErrInvalid    = vfs.EINVAL
	ErrClosed     = vfs.ECLOSED

	ErrDirNotEmpty    = vfs.ENOTEMPTY
	ErrIllegalSeek    = vfs.ESPIPE
	ErrBadFileDesc    = vfs.EBADF
	ErrReadOnly       = vfs.EROFS
	ErrNotImplemented = vfs.ENOSYS

	ErrNotDir = syscall.ENOTDIR
)

type (
	PathError = gofs.PathError
	LinkError = os.LinkError
)

func convertFsErr(err error) error {
	switch {
	case errors.Is(err, rfs.ErrorObjectNotFound):
		return ErrNotExist
	default:
		return err
	}
}

func pathError(op, path string, err error) error {
	var perr *gofs.PathError
	if errors.As(err, &perr) {
		return perr
	}

	return &gofs.PathError{Op: op, Path: path, Err: convertFsErr(err)}
}

func linkError(op, src, dst string, err error) error {
	var lerr *os.LinkError
	if errors.As(err, &lerr) {
		return lerr
	}

	var perr *gofs.PathError
	if errors.As(err, &perr) {
		return perr
	}

	return &LinkError{Op: op, Old: src, New: dst, Err: convertFsErr(err)}
}
