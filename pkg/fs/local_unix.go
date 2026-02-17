//go:build !windows

package fs

import (
	"os"
	"syscall"
)

func openFile(path string, flags Flags, perms FileMode) (File, error) {
	file, opErr := os.OpenFile(path, flags, perms)
	if opErr != nil {
		return nil, opErr //nolint:wrapcheck //no need to wrap here
	}

	if err := lockFile(file); err != nil {
		_ = file.Close()

		return nil, err
	}

	return file, nil
}

func lockFile(file *os.File) error {
	if info, err := file.Stat(); err != nil {
		return err //nolint:wrapcheck //no need to wrap here
	} else if info.IsDir() {
		return nil
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return pathError("lock", file.Name(), err)
	}

	return nil
}
