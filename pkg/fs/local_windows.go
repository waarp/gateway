package fs

import (
	"os"

	"golang.org/x/sys/windows"
)

func getAccess(flags Flags) uint32 {
	switch {
	case HasFlag(flags, FlagReadWrite):
		return windows.GENERIC_READ | windows.GENERIC_WRITE
	case HasFlag(flags, FlagWriteOnly):
		return windows.GENERIC_WRITE
	default:
		return windows.GENERIC_READ
	}
}

func getCreateMode(flags Flags) uint32 {
	switch {
	case HasFlag(flags, FlagCreate):
		if HasFlag(flags, FlagExclusive) {
			return windows.CREATE_NEW
		}

		return windows.OPEN_ALWAYS
	case HasFlag(flags, FlagTruncate):
		return windows.TRUNCATE_EXISTING
	default:
		return windows.OPEN_EXISTING
	}
}

func openFile(path string, flags Flags, mode FileMode) (File, error) {
	access := getAccess(flags)
	createMode := getCreateMode(flags)

	if access == windows.GENERIC_READ {
		//nolint:wrapcheck //no need to wrap here
		return os.Open(path)
	}

	if path == "" {
		return nil, pathError("open", path, ErrNotExist)
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
		if createMode == windows.OPEN_ALWAYS && HasFlag(flags, FlagTruncate) {
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
