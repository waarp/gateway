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

func parseBlobOpts(account, key string, confMap map[string]string) configmap.Simple {
	const (
		chunkSizeKey = "chunk_size"
		listChunkKey = "list_chunk"

		defaultChunkSize = "4MiB"
		defaultListChunk = "1000"
	)

	opts := parseAzureOpts(account, key, confMap)

	if opts[chunkSizeKey] == "" {
		opts[chunkSizeKey] = defaultChunkSize
	}

	if opts[listChunkKey] == "" {
		opts[listChunkKey] = defaultListChunk
	}

	return opts
}

func newBlobVFS(name, account, key, root string, confMap map[string]string) (*vfs.VFS, error) {
	opts := parseBlobOpts(account, key, confMap)
	vfsOpts := internal.VFSOpts()

	abfs, err := azureblob.NewFs(context.Background(), name, root, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure blob filesystem: %w", err)
	}

	return vfs.New(abfs, vfsOpts), nil
}

func NewBlobFS(name, account, key string, opts map[string]string) (fs.FS, error) {
	abvfs, err := newBlobVFS(name, account, key, "", opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: abvfs}, nil
}
