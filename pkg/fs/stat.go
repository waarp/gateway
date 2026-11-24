package fs

import (
	"errors"
	gofs "io/fs"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/internal/osfs"
)

var ErrNotLocalFS = errors.New("only local filesystems support extra file timestamps")

type FileInfo = gofs.FileInfo

func Stat(path string) (FileInfo, error) {
	parsed, srcFs, err := parseFs(path)
	if err != nil {
		return nil, err
	}

	info, err := srcFs.Stat(parsed.Path)
	if err != nil {
		return nil, pathError("stat", path, err)
	}

	return info, nil
}

func GetFileMTime(path string) (time.Time, error) {
	info, err := Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

func GetFileATime(raw string) (time.Time, error) {
	path, err := localParse(raw)
	if err != nil {
		return time.Time{}, err
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return osfs.GetFileATime(path)
}

func GetFileBTime(raw string) (time.Time, error) {
	path, err := localParse(raw)
	if err != nil {
		return time.Time{}, err
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return osfs.GetFileBTime(path)
}
