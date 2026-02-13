package fs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rclone/rclone/backend/local"
	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

var _ FS = &LocalFS{}

type LocalFS struct {
	locRFS rfs.Fs
}

//nolint:contextcheck //it's fine not to pass context here
func getLocalFs(parsed *parsedPath) (FS, error) {
	root := "/"
	if volume := filepath.VolumeName(parsed.Path); volume != "" {
		root = filepath.ToSlash(volume) + "/"
	}

	localRFS, err := local.NewFs(context.Background(), "local", root, configmap.Simple{})
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate local fs: %w", err)
	}

	return &LocalFS{locRFS: localRFS}, nil
}

func (l *LocalFS) fs() rfs.Fs { return l.locRFS }

func (l *LocalFS) Open(name string) (fs.File, error) {
	//nolint:wrapcheck,gosec //no need to wrap here
	return os.Open(name)
}

func (l *LocalFS) OpenFile(name string, flags Flags, perm FileMode) (File, error) {
	//nolint:gosec //file inclusion is checked elsewhere
	return os.OpenFile(name, flags, perm)
}

func (l *LocalFS) Stat(name string) (FileInfo, error) {
	//nolint:wrapcheck //no need to wrap here
	return os.Stat(name)
}

func (l *LocalFS) ReadDir(name string) ([]DirEntry, error) {
	//nolint:wrapcheck //no need to wrap here
	return os.ReadDir(name)
}

func (l *LocalFS) MkdirAll(path string, perm FileMode) error {
	//nolint:wrapcheck //no need to wrap here
	return os.MkdirAll(path, perm)
}

func (l *LocalFS) Rename(oldpath, newpath string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.Rename(oldpath, newpath)
}

func (l *LocalFS) Remove(path string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.Remove(path)
}

func (l *LocalFS) RemoveAll(path string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.RemoveAll(path)
}
