package fs

import (
	"errors"

	"github.com/hack-pad/hackpadfs"
)

var (
	ErrInvalid    = hackpadfs.ErrInvalid
	ErrPermission = hackpadfs.ErrPermission
	ErrExist      = hackpadfs.ErrExist
	ErrNotExist   = hackpadfs.ErrNotExist
	ErrClosed     = hackpadfs.ErrClosed

	ErrIsDir          = hackpadfs.ErrIsDir
	ErrNotDir         = hackpadfs.ErrNotDir
	ErrNotEmpty       = hackpadfs.ErrNotEmpty
	ErrNotImplemented = hackpadfs.ErrNotImplemented
)

type (
	LinkError = hackpadfs.LinkError
	PathError = hackpadfs.PathError
)

func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist)
}
