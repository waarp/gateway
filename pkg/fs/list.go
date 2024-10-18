package fs

import gofs "io/fs"

type DirEntry = gofs.DirEntry

func List(path string) ([]DirEntry, error) {
	parsed, srcFs, fsErr := parseFs(path)
	if fsErr != nil {
		return nil, fsErr
	}

	entries, listErr := srcFs.ReadDir(parsed.Path)
	if listErr != nil {
		return nil, pathError("list", path, listErr)
	}

	return entries, nil
}
