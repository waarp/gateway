package fs

import (
	"errors"
	"fmt"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rclone/rclone/fs"
)

var (
	ErrUnknownFS     = errors.New("unknown cloud instance")
	ErrUnknownFSType = errors.New("unknown cloud type")
)

//nolint:gochecknoglobals //global vars are needed here
var (
	FileSystems = xsync.NewMap[string, FS]()
	backends    = xsync.NewMap[string, Backend]()
)

type FSMaker func(name, key, secret string, opts map[string]string) (FS, error)

func Register(kind string, back Backend) {
	backends.Store(kind, back)
}

func Unregister(kind string) {
	backends.Delete(kind)
}

// Backend describes the storage of rclone a cloud type is built on: its name
// in the registry of rclone, whose options (with their defaults) are the ones
// an instance of the type accepts.
type Backend struct {
	Type    string
	Make    FSMaker
	Options fs.Options
}

func GetBackend(name string) (Backend, bool) {
	return backends.Load(name)
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
