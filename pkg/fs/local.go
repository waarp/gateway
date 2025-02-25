package fs

import (
	"context"
	"errors"
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
	root        string
	openedFiles []*os.File
	locRFS      rfs.Fs
}

func GetTestFS(root string) (*LocalFS, error) {
	localFS, err := local.NewFs(context.Background(), "testfs", root, configmap.Simple{})
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate local fs: %w", err)
	}

	return &LocalFS{
		root:        root,
		openedFiles: []*os.File{},
		locRFS:      localFS,
	}, nil
}

//nolint:contextcheck //it's fine not to pass context here
func getLocalFs(parsed *parsedPath) (FS, error) {
	root := "/"
	if volume := filepath.VolumeName(parsed.Path); volume != "" {
		root = volume + "/"
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
	return os.Open(filepath.Join(l.root, name))
}

func (l *LocalFS) OpenFile(name string, flags Flags, perm FileMode) (File, error) {
	//nolint:gosec //file inclusion is checked elsewhere
	file, err := os.OpenFile(filepath.Join(l.root, name), flags, perm)

	if file != nil && l.openedFiles != nil {
		l.openedFiles = append(l.openedFiles, file)
	}

	//nolint:wrapcheck //no need to wrap here
	return file, err
}

func (l *LocalFS) Stat(name string) (FileInfo, error) {
	//nolint:wrapcheck //no need to wrap here
	return os.Stat(filepath.Join(l.root, name))
}

func (l *LocalFS) ReadDir(name string) ([]DirEntry, error) {
	//nolint:wrapcheck //no need to wrap here
	return os.ReadDir(filepath.Join(l.root, name))
}

func (l *LocalFS) MkdirAll(path string, perm FileMode) error {
	//nolint:wrapcheck //no need to wrap here
	return os.MkdirAll(filepath.Join(l.root, path), perm)
}

func (l *LocalFS) Rename(oldpath, newpath string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.Rename(filepath.Join(l.root, oldpath), filepath.Join(l.root, newpath))
}

func (l *LocalFS) Remove(path string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.Remove(filepath.Join(l.root, path))
}

func (l *LocalFS) RemoveAll(path string) error {
	//nolint:wrapcheck //no need to wrap here
	return os.RemoveAll(filepath.Join(l.root, path))
}

func (l *LocalFS) Flush() error {
	var errs []error

	for _, f := range l.openedFiles {
		if err := f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			errs = append(errs, err)
		}
	}

	l.openedFiles = nil

	return errors.Join(errs...)
}
