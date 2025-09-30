package fs

import (
	"os"

	"golang.org/x/sys/windows"
)

const (
	reserved = 0
	allBytes = ^uint32(0)
)

//nolint:wrapcheck //no need to wrap here
func lockFile(file *os.File) error {
	ol := new(windows.Overlapped)

	if err := windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK, reserved, allBytes, allBytes, ol); err != nil {
		return &PathError{Op: "Lock", Path: file.Name(), Err: err}
	}

	return nil
}

//nolint:wrapcheck //no need to wrap here
func lockFileR(file *os.File) error {
	ol := new(windows.Overlapped)

	if err := windows.LockFileEx(windows.Handle(file.Fd()),
		0, reserved, allBytes, allBytes, ol); err != nil {
		return &PathError{Op: "RLock", Path: file.Name(), Err: err}
	}

	return nil
}

//nolint:wrapcheck //no need to wrap here
func unlockFile(file *os.File) error {
	ol := new(windows.Overlapped)

	if err := windows.UnlockFileEx(windows.Handle(file.Fd()), 0, allBytes, allBytes, ol); err != nil {
		return &PathError{Op: "Unlock", Path: file.Name(), Err: err}
	}

	return nil
}
