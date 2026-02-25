package protoutils

import (
	"errors"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

var (
	ErrReadOnDir  = errors.New("cannot read on directory")
	ErrWriteOnDir = errors.New("cannot write on directory")
	ErrSeekOnDir  = errors.New("cannot seek on directory")
)

type FakeDir struct {
	name     string
	children FakeDirInfos
}

func (f *FakeDir) Stat() (fs.FileInfo, error) {
	return FakeDirInfo(f.name), nil
}

func readDir[T any](n int, children []T) ([]T, error) {
	if n <= 0 {
		return children, nil
	}

	if n >= len(children) {
		return children, io.EOF
	}

	return children[:n], nil
}

func (*FakeDir) Read([]byte) (int, error)           { return 0, ErrReadOnDir }
func (*FakeDir) Write([]byte) (int, error)          { return 0, ErrWriteOnDir }
func (*FakeDir) ReadAt([]byte, int64) (int, error)  { return 0, ErrReadOnDir }
func (*FakeDir) WriteAt([]byte, int64) (int, error) { return 0, ErrWriteOnDir }
func (*FakeDir) Seek(int64, int) (int64, error)     { return 0, ErrSeekOnDir }

func (*FakeDir) Sync() error  { return nil }
func (*FakeDir) Close() error { return nil }

func (f *FakeDir) Readdir(n int) ([]fs.FileInfo, error) {
	return readDir(n, f.children.AsFileInfos())
}

func (f *FakeDir) ReadDir(n int) ([]fs.DirEntry, error) {
	return readDir(n, f.children.AsDirEntries())
}
