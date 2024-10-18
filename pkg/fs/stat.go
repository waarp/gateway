package fs

import (
	gofs "io/fs"
)

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
