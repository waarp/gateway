package fs

func MkdirAll(path string) error {
	const mkdirPerms FileMode = 0o740

	parsed, srcFs, parsErr := parseFs(path)
	if parsErr != nil {
		return parsErr
	}

	if err := srcFs.MkdirAll(parsed.Path, mkdirPerms); err != nil {
		return pathError("mkdir", path, err)
	}

	return nil
}
