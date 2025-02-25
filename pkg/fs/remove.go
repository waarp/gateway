package fs

func Remove(path string) error {
	parsed, srcFs, err := parseFs(path)
	if err != nil {
		return err
	}

	if rmErr := srcFs.Remove(parsed.Path); rmErr != nil {
		return pathError("remove", path, rmErr)
	}

	return nil
}

func RemoveAll(path string) error {
	parsed, srcFs, err := parseFs(path)
	if err != nil {
		return err
	}

	return removeAll(srcFs, parsed.Path)
}

func removeAll(srcFs FS, path string) error {
	if rmAllFs, ok := srcFs.(interface{ RemoveAll(path string) error }); ok {
		if err := rmAllFs.RemoveAll(path); err != nil {
			return pathError("removeall", path, err)
		}

		return nil
	}

	fullPath := srcFs.fs().Name() + ":" + path

	stat, statErr := srcFs.Stat(path)
	if statErr != nil {
		return pathError("stat", fullPath, statErr)
	}

	if stat.IsDir() {
		entries, listErr := srcFs.ReadDir(path)
		if listErr != nil {
			return pathError("list", fullPath, listErr)
		}

		for _, entry := range entries {
			if err := removeAll(srcFs, entry.Name()); err != nil {
				return err
			}
		}
	}

	if err := srcFs.Remove(path); err != nil {
		return pathError("remove", fullPath, err)
	}

	return nil
}
