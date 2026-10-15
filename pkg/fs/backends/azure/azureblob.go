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

func parseBlobOpts(account, key string, opts map[string]string) configmap.Simple {
	const (
		chunkSizeKey = "chunk_size"
		listChunkKey = "list_chunk"

		defaultChunkSize = "4MiB"
		defaultListChunk = "1000"
	)

	config := parseAzureOpts(account, key, opts)

	if config[chunkSizeKey] == "" {
		config[chunkSizeKey] = defaultChunkSize
	}

	if config[listChunkKey] == "" {
		config[listChunkKey] = defaultListChunk
	}

	return config
}

func newBlobVFS(name, account, key, root string, opts map[string]string) (*vfs.VFS, error) {
	config, err := internal.WithDefaults("azureblob", parseBlobOpts(account, key, opts))
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure blob filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()

	abfs, err := azureblob.NewFs(context.Background(), name, root, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure blob filesystem: %w", err)
	}

	return vfs.New(context.Background(), abfs, vfsOpts), nil
}

func NewBlobFS(name, account, key string, opts map[string]string) (fs.FS, error) {
	abvfs, err := newBlobVFS(name, account, key, "", opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: abvfs}, nil
}
