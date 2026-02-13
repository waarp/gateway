//go:build !windows

package fs

import (
	"os"
	"syscall"
)

func openFile(path string, flags Flags, perms FileMode) (File, error) {
	file, opErr := os.OpenFile(path, flags, perms)
	if opErr != nil {
		//nolint:wrapcheck //ne need to wrap here
		return nil, opErr
	}

	// Try to acquire an exclusive lock immediately.
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()

		return nil, &os.PathError{Op: "flock", Path: path, Err: err}
	}

	return file, nil
}
