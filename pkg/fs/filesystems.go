package fs

import (
	"errors"
	"fmt"

	"github.com/puzpuzpuz/xsync/v4"
)

var (
	ErrUnknownFS     = errors.New("unknown cloud instance")
	ErrUnknownFSType = errors.New("unknown cloud type")
)

//nolint:gochecknoglobals //global vars are needed here
var (
	FileSystems = xsync.NewMap[string, FS]()
	fsMakers    = xsync.NewMap[string, FSMaker]()
	libraries   = xsync.NewMap[string, Library]()
)

type FSMaker func(name, key, secret string, opts map[string]string) (FS, error)

func Register(kind string, mkfs FSMaker) {
	fsMakers.Store(kind, mkfs)
}

func Unregister(kind string) {
	fsMakers.Delete(kind)
}

// Library describes the storage of rclone a cloud type is built on: its name
// in the registry of rclone, whose options (with their defaults) are the ones
// an instance of the type accepts. The web interface reads it to list these
// options and to refuse the others.
type Library struct {
	Name string
}

// Describe records the library a cloud type is built on.
func Describe(kind string, lib Library) {
	libraries.Store(kind, lib)
}

// LibraryOf returns the library a cloud type is built on, when its backend
// described it.
func LibraryOf(kind string) (Library, bool) {
	return libraries.Load(kind)
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
