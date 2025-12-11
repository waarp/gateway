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

const bucketKey = "bucket"

var ErrMissingBucket = errors.New("no GCS bucket specified")

func parseOpts(key, secret string, opts map[string]string) (configmap.Simple, error) {
	opts["directory_markers"] = "true"

	const (
		credKey     = "service_account_credentials"
		credFileKey = "service_account_file"
	)

	if key != "" {
		opts[credFileKey] = key
	}

	if secret != "" {
		opts[credKey] = secret
	}

	if bucket := opts[bucketKey]; bucket == "" {
		return nil, ErrMissingBucket
	}

	return opts, nil
}

func newVFS(name, key, secret string, confMap map[string]string) (*vfs.VFS, error) {
	opts, err := parseOpts(key, secret, confMap)
	if err != nil {
		return nil, err
	}

	root := opts[bucketKey]

	s3fs, err := gcs.NewFs(context.Background(), name, root, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()
	s3vfs := vfs.New(s3fs, vfsOpts)

	return s3vfs, nil
}

func NewFS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	gcvfs, err := newVFS(name, key, secret, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: gcvfs}, nil
}
