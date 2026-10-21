package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/rclone/rclone/backend/azurefiles"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

var ErrMissingFilesShare = errors.New("no Azure Files share specified")

// FilesStorage is the storage of rclone the Azure Files backend is built on.
const FilesStorage = "azurefiles"

func parseFilesOpts(account, key string, userOpts map[string]string) (configmap.Mapper, error) {
	// Credential options
	credOpts := azureCredOpts(account, key)

	// Make config map
	config := internal.MakeOptions(FilesStorage, userOpts, credOpts)

	if share, _ := config.Get("share_name"); share == "" {
		return nil, ErrMissingFilesShare
	}

	return config, nil
}

func newFilesVFS(name, account, key string, opts map[string]string) (*vfs.VFS, error) {
	config, err := parseFilesOpts(account, key, opts)
	if err != nil {
		return nil, err
	}

	vfsOpts := internal.VFSOpts()

	affs, err := azurefiles.NewFs(context.Background(), name, "", config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure files filesystem: %w", err)
	}

	return vfs.New(context.Background(), affs, vfsOpts), nil
}

func NewFilesFS(name, account, key string, opts map[string]string) (fs.FS, error) {
	afvfs, err := newFilesVFS(name, account, key, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: afvfs}, nil
}
