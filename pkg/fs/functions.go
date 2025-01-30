package fs

import (
	"fmt"
	gofs "io/fs"
	"path/filepath"
	"runtime"

	hfs "github.com/hack-pad/hackpadfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	DefaultFilePerm = 0o640
	DefaultDirPerm  = 0o740
)

// OpenFile opens and returns the given file with the given flags and permissions.
func OpenFile(fs FS, name *types.FSPath, flag int, perm FileMode) (File, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.OpenFile(fs, name.FSPath(), flag, perm)
}

// Open opens and returns the given file in read-only mode.
func Open(fs FS, name *types.FSPath) (File, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return fs.Open(name.FSPath())
}

// Create creates and returns the given file. If the file already exists, it is
// truncated.
func Create(fs FS, name *types.FSPath) (File, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return OpenFile(fs, name, FlagRW|FlagCreate|FlagTruncate, DefaultFilePerm)
}

// ReadFile reads the whole content of the given file, and returns it.
func ReadFile(fs FS, name *types.FSPath) ([]byte, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.ReadFile(fs, name.FSPath())
}

// WriteFullFile writes the given content to the given file. If the file does not
// exist, it is created. If the file does exist, it is truncated, and its content
// overwritten.
func WriteFullFile(fs FS, name *types.FSPath, data []byte) error {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.WriteFullFile(fs, name.FSPath(), data, DefaultFilePerm)
}

// MkdirAll creates the given directory and all of its parents (if they don't
// already exist).
func MkdirAll(fs FS, name *types.FSPath) error {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.MkdirAll(fs, name.FSPath(), DefaultDirPerm)
}

// Stat returns a FileInfo describing the given file.
func Stat(fs FS, name *types.FSPath) (FileInfo, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.Stat(fs, name.FSPath())
}

// Rename renames (moves) oldname to newname. If newname already exists and is
// not a directory, Rename replaces it.
func Rename(fs FS, oldname, newname *types.FSPath) error {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.Rename(fs, oldname.FSPath(), newname.FSPath())
}

// Remove removes the named file or directory.
func Remove(fs FS, name *types.FSPath) error {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.Remove(fs, name.FSPath())
}

// RemoveAll removes the named file or directory and all of its children.
func RemoveAll(fs FS, name *types.FSPath) error {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.RemoveAll(fs, name.FSPath())
}

func ReadDir(fs FS, name *types.FSPath) ([]DirEntry, error) {
	//nolint:wrapcheck //wrapping adds nothing here
	return hfs.ReadDir(fs, name.FSPath())
}

func Glob(fs FS, pattern *types.FSPath) ([]*types.FSPath, error) {
	matches, globErr := gofs.Glob(fs, pattern.FSPath())
	if globErr != nil {
		return nil, fmt.Errorf("failed to walk the dir: %w", globErr)
	}

	paths := make([]*types.FSPath, len(matches))
	root := pattern

	if runtime.GOOS == "windows" {
		root.Path = filepath.VolumeName(root.Path)
	} else {
		root.Path = "/"
	}

	for i, match := range matches {
		paths[i] = root.JoinPath(match)
	}

	return paths, nil
}
