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

func IsOnSameFS(path1, path2 *types.FSPath) bool {
	if path1.Backend == "" && path2.Backend == "" {
		return filepath.VolumeName(path1.Path) == filepath.VolumeName(path2.Path)
	}

	return path1.Backend == path2.Backend
}

func GetFileSystem(db database.ReadAccess, path *types.FSPath) (fs.FS, error) {
	if path.Backend == "" || path.Backend == "file" {
		return NewLocalFS(path.Path)
	}

	if testFS, ok := filesystems.TestFileSystems.Load(path.Backend); ok {
		return testFS, nil
	}

	var cloud model.CloudInstance
	if err := db.Get(&cloud, "name=?", path.Backend).And("owner=?",
		conf.GlobalConfig.GatewayName).Run(); database.IsNotFound(err) {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCloudInstance, path.Backend)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud instance %q: %w", path.Backend, err)
	}

	mkfs, ok := filesystems.FileSystems.Load(cloud.Type)
	if !ok {
		return nil, fmt.Errorf("%w %q", filesystems.ErrUnknownFileSystem, cloud.Type)
	}

	return mkfs(cloud.Key, string(cloud.Secret), cloud.Options)
}
