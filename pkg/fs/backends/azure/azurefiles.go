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

const shareNameKey = "share_name"

func parseFilesOpts(account, key string, confMap map[string]string) (configmap.Simple, error) {
	opts := parseAzureOpts(account, key, confMap)

	if share := opts[shareNameKey]; share == "" {
		return nil, ErrMissingFilesShare
	}

	return opts, nil
}

func newFilesVFS(name, account, key string, confMap map[string]string) (*vfs.VFS, error) {
	opts, err := parseFilesOpts(account, key, confMap)
	if err != nil {
		return nil, err
	}

	vfsOpts := internal.VFSOpts()

	affs, err := azurefiles.NewFs(context.Background(), name, "", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate azure files filesystem: %w", err)
	}

	return vfs.New(affs, vfsOpts), nil
}

func NewFilesFS(name, account, key string, opts map[string]string) (fs.FS, error) {
	afvfs, err := newFilesVFS(name, account, key, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: afvfs}, nil
}
