package gcs

import (
	"context"
	"errors"
	"fmt"

	gcs "github.com/rclone/rclone/backend/googlecloudstorage"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

var ErrMissingBucket = errors.New("no GCS bucket specified")

func parseOpts(accFile, accCreds string, userOpts map[string]string,
) (config configmap.Mapper, root string, err error) {
	// Credential options
	overrides := configmap.Simple{
		"service_account_credentials": accCreds,
		"service_account_file":        accFile,
	}

	// Make config map
	config = internal.MakeOptions("gcs", userOpts, overrides)

	// Check mandatory options
	if bucket, ok := config.Get("bucket"); !ok || bucket == "" {
		return nil, "", ErrMissingBucket
	} else {
		root = bucket
	}

	// Restore old defaults
	if _, ok := userOpts["directory_markers"]; !ok {
		overrides.Set("directory_markers", "true")
	}

	return config, root, nil
}

func newVFS(name, accFile, accCreds string, opts map[string]string) (*vfs.VFS, error) {
	config, root, err := parseOpts(accFile, accCreds, opts)
	if err != nil {
		return nil, err
	}

	gcfs, err := gcs.NewFs(context.Background(), name, root, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate GCS filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()

	return vfs.New(context.Background(), gcfs, vfsOpts), nil
}

func NewFS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	gcvfs, err := newVFS(name, key, secret, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: gcvfs}, nil
}
