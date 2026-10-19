package azure

import (
	"context"
	"fmt"

	"github.com/rclone/rclone/backend/azureblob"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

func parseBlobOpts(account, key string, userOpts map[string]string) configmap.Mapper {
	// Credential options
	credOpts := azureCredOpts(account, key)

	// Make config map
	config := internal.MakeOptions("azureblob", userOpts, credOpts)

	return config
}

func newBlobVFS(name, account, key string, opts map[string]string) (*vfs.VFS, error) {
	config := parseBlobOpts(account, key, opts)

	vfsOpts := internal.VFSOpts()

	abfs, err := azureblob.NewFs(context.Background(), name, "", config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure blob filesystem: %w", err)
	}

	return vfs.New(context.Background(), abfs, vfsOpts), nil
}

func NewBlobFS(name, account, key string, opts map[string]string) (fs.FS, error) {
	abvfs, err := newBlobVFS(name, account, key, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: abvfs}, nil
}
