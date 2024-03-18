// Package filesystems contains a list of all the known file systems.
package filesystems

import (
	"errors"
	"io/fs"
)

const FileScheme = "file"

var ErrUnknownFileSystem = errors.New("unknown file system")

//nolint:gochecknoglobals //global vars are required here
var (
	// TestFileSystems contains a list of all the test file systems. DO NOT USE
	// OUTSIDE OF TESTS.
	TestFileSystems = map[string]fs.FS{}

	// FileSystems contains a list of all the known file systems. The map associates
	// the file system's scheme with a function to instantiate the file system.
	FileSystems = map[string]func(key, secret string, options map[string]any) (fs.FS, error){}
)

func DoesFileSystemExist(scheme string) bool {
	return scheme == FileScheme || TestFileSystems[scheme] != nil || FileSystems[scheme] != nil
}
