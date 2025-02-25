package fs

import (
	"errors"
	"fmt"

	"github.com/puzpuzpuz/xsync"
)

var (
	ErrUnknownFS     = errors.New("unknown cloud instance")
	ErrUnknownFSType = errors.New("unknown cloud type")
)

//nolint:gochecknoglobals //global vars are needed here
var (
	FileSystems = xsync.NewMapOf[FS]()
	fsMakers    = xsync.NewMapOf[FSMaker]()
)

type FSMaker func(name, key, secret string, opts map[string]string) (FS, error)

func Register(kind string, mkfs FSMaker) {
	fsMakers.Store(kind, mkfs)
}

func Unregister(kind string) {
	fsMakers.Delete(kind)
}

func getFS(parsed *parsedPath) (FS, error) {
	if parsed.Name == "" {
		return getLocalFs(parsed)
	}

	if fsys, ok := FileSystems.Load(parsed.Name); ok {
		return fsys, nil
	}

	return nil, fmt.Errorf("%w %q", ErrUnknownFS, parsed.Name)
}
