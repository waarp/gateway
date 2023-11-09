// Package fs is the package used for managing transfer files in a file system
// agnostic way.
package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var ErrUnknownCloudInstance = errors.New("unknown cloud instance")

func IsOnSameFS(path1, path2 *types.URL) bool {
	if path1.Scheme == filesystems.FileScheme && path2.Scheme == filesystems.FileScheme {
		osPath1 := path1.OSPath()
		osPath2 := path2.OSPath()

		return filepath.VolumeName(osPath1) == filepath.VolumeName(osPath2)
	}

	return path1.Scheme == path2.Scheme && path1.Host == path2.Host
}

func GetFileSystem(db database.ReadAccess, url *types.URL) (fs.FS, error) {
	if url.Scheme == filesystems.FileScheme {
		return NewLocalFS(url.OSPath())
	}

	if testFS, ok := filesystems.TestFileSystems[url.Scheme]; ok {
		return testFS, nil
	}

	mkfs := filesystems.FileSystems[url.Scheme]
	if mkfs == nil {
		return nil, fmt.Errorf("%w %q", filesystems.ErrUnknownFileSystem, url.Scheme)
	}

	var cloud model.CloudInstance
	if err := db.Get(&cloud, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
		url.Host).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %q", ErrUnknownCloudInstance, url.Host)
		}

		return nil, fmt.Errorf("failed to retrieve cloud instance %q: %w", url.Host, err)
	}

	return mkfs(cloud.Key, string(cloud.Secret), cloud.Options)
}
