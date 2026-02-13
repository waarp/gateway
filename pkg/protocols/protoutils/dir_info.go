package protoutils

import (
	gofs "io/fs"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

const fileMode = 0o777

// FakeDirInfo is an implementation of fs.FileInfo for the virtual directories.
type FakeDirInfo string

func (f FakeDirInfo) Type() gofs.FileMode          { return f.Mode() }
func (f FakeDirInfo) Info() (gofs.FileInfo, error) { return f, nil }
func (f FakeDirInfo) Name() string                 { return string(f) }
func (FakeDirInfo) Size() int64                    { return 0 }
func (FakeDirInfo) Mode() gofs.FileMode            { return gofs.ModeDir | fileMode }
func (FakeDirInfo) ModTime() time.Time             { return time.Now() }
func (FakeDirInfo) IsDir() bool                    { return true }

type FakeDirInfos []FakeDirInfo

func (f FakeDirInfos) AsFileInfos() []fs.FileInfo {
	fileInfos := make([]fs.FileInfo, len(f))
	for i, elem := range f {
		fileInfos[i] = elem
	}

	return fileInfos
}

func (f FakeDirInfos) AsDirEntries() []fs.DirEntry {
	dirEntries := make([]fs.DirEntry, len(f))
	for i, elem := range f {
		dirEntries[i] = elem
	}

	return dirEntries
}
