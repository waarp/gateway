// Package filesystems contains a list of all the known file systems.
package filesystems

import (
	"errors"
	"io/fs"

	"github.com/puzpuzpuz/xsync"
)

const FileScheme = "file"

var ErrUnknownFileSystem = errors.New("unknown file system")

type FSMaker func(key, secret string, options map[string]any) (fs.FS, error)

//nolint:gochecknoglobals //global vars are required here
var (
	// TestFileSystems contains a list of all the test file systems. DO NOT USE
	// OUTSIDE OF TESTS.
	TestFileSystems = xsync.NewMapOf[fs.FS]()

	// FileSystems contains a list of all the known file systems. The map associates
	// the file system's scheme with a function to instantiate the file system.
	FileSystems = xsync.NewMapOf[FSMaker]()
)

func DoesFileSystemExist(scheme string) bool {
	_, tfsOK := TestFileSystems.Load(scheme)
	_, fsOK := FileSystems.Load(scheme)

	return scheme == FileScheme || tfsOK || fsOK
}
