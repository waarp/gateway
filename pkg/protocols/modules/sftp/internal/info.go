// Package internal contains a few internal functions & structs for the sftp module.
package internal

import (
	"os"
	"time"
)

const fileMode = 0o777

// DirInfo is an implementation of os.FileInfo for the SFTP server's virtual directories.
type DirInfo struct{ Dir string }

// Name returns the directory's path.
func (d *DirInfo) Name() string {
	return d.Dir
}

// Size returns the directory's size (which is always 0 for directories).
func (d *DirInfo) Size() int64 {
	return 0
}

// Mode returns the directory's Filemode.
func (d *DirInfo) Mode() os.FileMode {
	return os.ModeDir | fileMode
}

// ModTime returns the directory's last modification time (which is always now).
func (d *DirInfo) ModTime() time.Time {
	return time.Now()
}

// IsDir returns whether the directory is a directory (it always is).
func (d *DirInfo) IsDir() bool {
	return true
}
