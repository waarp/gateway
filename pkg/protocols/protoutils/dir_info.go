package protoutils

import (
	gofs "io/fs"
	"time"
)

const fileMode = 0o777

// FakeDirInfo is an implementation of fs.FileInfo for the virtual directories.
type FakeDirInfo string

func (f FakeDirInfo) Name() string      { return string(f) }
func (FakeDirInfo) Size() int64         { return 0 }
func (FakeDirInfo) Mode() gofs.FileMode { return gofs.ModeDir | fileMode }
func (FakeDirInfo) ModTime() time.Time  { return time.Now() }
func (FakeDirInfo) IsDir() bool         { return true }
