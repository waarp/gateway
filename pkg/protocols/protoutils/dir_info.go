package protoutils

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

const fileMode = 0o777

// FakeDirInfo is an implementation of fs.FileInfo for the virtual directories.
type FakeDirInfo string

func (f FakeDirInfo) Name() string     { return string(f) }
func (FakeDirInfo) Size() int64        { return 0 }
func (FakeDirInfo) Mode() fs.FileMode  { return fs.ModeDir | fileMode }
func (FakeDirInfo) ModTime() time.Time { return time.Now() }
func (FakeDirInfo) IsDir() bool        { return true }
