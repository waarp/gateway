// Package internal contains a few internal functions & structs for the sftp module.
package internal

import (
	"os"
	"time"
)

// DirInfo is an implementation of os.FileInfo for the SFTP server's virtual directories.
type DirInfo string

// Name returns the directory's path.
func (d DirInfo) Name() string {
	return string(d)
}

// Size returns the directory's size (which is always 0 for directories).
func (d DirInfo) Size() int64 {
	return 0
}

// Mode returns the directory's Filemode.
func (d DirInfo) Mode() os.FileMode {
	return os.ModeDir | 0o700
}

// ModTime returns the directory's last modification time (which is always now).
func (d DirInfo) ModTime() time.Time {
	return time.Now()
}

// IsDir returns whether the directory is a directory (is always is).
func (d DirInfo) IsDir() bool {
	return true
}

// Sys returns additional info on the directory (there is never any).
func (d DirInfo) Sys() interface{} {
	return nil
}
