package internal

import (
	"os"
	"time"
)

type DirInfo string

func (d DirInfo) Name() string {
	return string(d)
}

func (d DirInfo) Size() int64 {
	return 0
}

func (d DirInfo) Mode() os.FileMode {
	return os.ModeDir | 0o700
}

func (d DirInfo) ModTime() time.Time {
	return time.Now()
}

func (d DirInfo) IsDir() bool {
	return true
}

func (d DirInfo) Sys() interface{} {
	return nil
}
