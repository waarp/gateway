// Package s3 provides a filesystem implementation for Amazon S3.
package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

var ErrMissingBucket = errors.New("no S3 bucket specified")

func parseOpts(key, secret string, userOpts map[string]string,
) (config configmap.Mapper, root string, err error) {
	// Credential options
	credOpts := configmap.Simple{
		"access_key_id":     key,
		"secret_access_key": secret,
	}

	// Make config map
	config = internal.MakeOptions("s3", userOpts, credOpts)

	// Check mandatory options
	if bucket, _ := config.Get("bucket"); bucket == "" {
		return nil, "", ErrMissingBucket
	}

	if isPathStyle, _ := config.Get("force_path_style"); isPathStyle == "true" {
		root, _ = config.Get("bucket")
	}

	return config, root, nil
}

func newVFS(name, key, secret string, opts map[string]string) (*vfs.VFS, error) {
	config, root, err := parseOpts(key, secret, opts)
	if err != nil {
		return nil, err
	}

	s3fs, err := s3.NewFs(context.Background(), name, root, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()
	s3vfs := vfs.New(context.Background(), s3fs, vfsOpts)

	return s3vfs, nil
}

func NewFS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	s3vfs, err := newVFS(name, key, secret, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: s3vfs}, nil
}
