// Package fs provides a filesystem abstraction over the rclone cloud connectors.
package fs

import (
	"fmt"
	gofs "io/fs"

	rfs "github.com/rclone/rclone/fs"
)

type FS interface {
	Open(name string) (gofs.File, error)
	OpenFile(name string, flags Flags, perm FileMode) (File, error)
	Stat(name string) (FileInfo, error)
	ReadDir(name string) ([]DirEntry, error)
	MkdirAll(path string, perm FileMode) error
	Rename(oldpath, newpath string) error
	Remove(path string) error

	fs() rfs.Fs
}

func newFS(name, kind, key, secret string, opts map[string]string) (FS, error) {
	mkfs, ok := fsMakers.Load(kind)
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownFSType, kind)
	}

	return mkfs(name, key, secret, opts)
}

func ValidateConfig(name, kind, key, secret string, opts map[string]string) error {
	_, err := newFS(name, kind, key, secret, opts)

	return err
}

func IsLocalPath(path string) bool {
	parsed, err := parsePath(path)
	if err != nil {
		return false
	}

	return parsed.Name == ""
}
