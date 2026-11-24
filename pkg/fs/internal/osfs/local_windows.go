package osfs

import (
	"io/fs"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func getAccess(flags int) uint32 {
	switch {
	case flags&os.O_RDWR != 0:
		return windows.GENERIC_READ | windows.GENERIC_WRITE
	case flags&os.O_WRONLY != 0:
		return windows.GENERIC_WRITE
	default:
		return windows.GENERIC_READ
	}
}

func getCreateMode(flags int) uint32 {
	switch {
	case flags&os.O_CREATE != 0:
		if flags&os.O_EXCL != 0 {
			return windows.CREATE_NEW
		}

		return windows.OPEN_ALWAYS
	case flags&os.O_TRUNC != 0:
		return windows.TRUNCATE_EXISTING
	default:
		return windows.OPEN_EXISTING
	}
}

func OpenFile(path string, flags int, mode fs.FileMode) (*os.File, error) {
	access := getAccess(flags)
	createMode := getCreateMode(flags)

	if access == windows.GENERIC_READ {
		//nolint:wrapcheck //no need to wrap here
		return os.Open(path)
	}

	if path == "" {
		return nil, pathError("open", path, os.ErrNotExist)
	}

	pathp, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, pathError("open", path, err)
	}

	handle, err := windows.CreateFile(
		pathp,
		access,
		0, // 0 = Exclusive Access
		nil,
		createMode,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err == nil {
		file := os.NewFile(uintptr(handle), path)

		// OPEN_ALWAYS keeps the content of an existing file: it is truncated
		// afterwards, as os.OpenFile does (see syscall.Open).
		if createMode == windows.OPEN_ALWAYS && (flags&os.O_TRUNC != 0) {
			if truncErr := file.Truncate(0); truncErr != nil {
				_ = file.Close()

				return nil, truncErr //nolint:wrapcheck //no need to wrap here
			}
		}

		return file, nil
	}

	// Fallback to os.OpenFile if CreateFile failed in case the target FS does
	// not support excusive access.
	return os.OpenFile(path, flags, mode) //nolint:wrapcheck //no need to wrap here
}

func GetFileATime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err //nolint:wrapcheck //wrapping adds nothing here
	}

	//nolint:forcetypeassert //always succeeds, guaranteed by build tag
	stat := info.Sys().(*syscall.Win32FileAttributeData)

	return time.Unix(0, stat.LastAccessTime.Nanoseconds()), nil
}

func GetFileBTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err //nolint:wrapcheck //wrapping adds nothing here
	}

	//nolint:forcetypeassert //always succeeds, guaranteed by build tag
	stat := info.Sys().(*syscall.Win32FileAttributeData)

	return time.Unix(0, stat.CreationTime.Nanoseconds()), nil
}
