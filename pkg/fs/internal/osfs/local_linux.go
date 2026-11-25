package osfs

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func GetFileATime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err //nolint:wrapcheck //wrapping adds nothing here
	}

	//nolint:forcetypeassert //always succeeds, guaranteed by build condition
	stat := info.Sys().(*syscall.Stat_t)

	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec), nil
}

func GetFileBTime(path string) (time.Time, error) {
	var statx unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT,
		unix.STATX_BTIME, &statx); err != nil {
		return time.Time{}, pathError("stat", path, err)
	}

	return time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)), nil
}
