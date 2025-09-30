//go:build !windows

package fs

import (
	"os"
	"syscall"
)

//nolint:wrapcheck //no need to wrap here
func lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

//nolint:wrapcheck //no need to wrap here
func lockFileR(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_SH)
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
