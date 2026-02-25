package protoutils

import (
	"errors"
	"io"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

var ErrSeekOutsideFile = errors.New("cannot seek outside file")

type FakeFile string

func (f FakeFile) Stat() (fs.FileInfo, error)       { return FakeFileInfo(f), nil }
func (FakeFile) Read([]byte) (int, error)           { return 0, io.EOF }
func (FakeFile) ReadAt([]byte, int64) (int, error)  { return 0, io.EOF }
func (FakeFile) Write([]byte) (int, error)          { return 0, io.EOF }
func (FakeFile) WriteAt([]byte, int64) (int, error) { return 0, io.EOF }
func (FakeFile) Close() error                       { return nil }
func (FakeFile) ReadDir(int) ([]fs.DirEntry, error) { return nil, fs.ErrNotDir }
func (FakeFile) Readdir(int) ([]fs.FileInfo, error) { return nil, fs.ErrNotDir }
func (FakeFile) Sync() error                        { return nil }
func (FakeFile) Seek(offset int64, _ int) (int64, error) {
	if offset == 0 {
		return 0, nil
	}

	return 0, ErrSeekOutsideFile
}

type FakeFileInfo string

func (f FakeFileInfo) Name() string       { return string(f) }
func (f FakeFileInfo) Size() int64        { return 0 }
func (f FakeFileInfo) Mode() fs.FileMode  { return fileMode }
func (f FakeFileInfo) ModTime() time.Time { return time.Now() }
func (f FakeFileInfo) IsDir() bool        { return false }
func (f FakeFileInfo) Sys() any           { return nil }
