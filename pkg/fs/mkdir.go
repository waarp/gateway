package fs

func MkdirAll(path string) error {
	parsed, srcFs, parsErr := parseFs(path)
	if parsErr != nil {
		return parsErr
	}

	if err := srcFs.MkdirAll(parsed.Path, DirPerms); err != nil {
		return pathError("mkdir", path, err)
	}

	return nil
}
