package fs

import "code.waarp.fr/apps/gateway/gateway/pkg/conf"

func MkdirAll(path string) error {
	const mkdirPerms FileMode = conf.DefaultDirPermissions

	parsed, srcFs, parsErr := parseFs(path)
	if parsErr != nil {
		return parsErr
	}

	if err := srcFs.MkdirAll(parsed.Path, mkdirPerms); err != nil {
		return pathError("mkdir", path, err)
	}

	return nil
}
