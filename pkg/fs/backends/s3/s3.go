// Package s3 provides a filesystem implementation for Amazon S3.
package s3

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

var ErrMissingBucket = errors.New("no S3 bucket specified")

const bucketKey = "bucket"

func parseOpts(key, secret string, opts map[string]string) (configmap.Simple, error) {
	// constant options
	const (
		dirMarkersKey  = "directory_markers"
		listVersionKey = "list_version"
		listChunkKey   = "list_chunk"

		dirMarkers  = "true"
		listVersion = "2"
		listChunk   = "1000"

		accessIDKey     = "access_key_id"
		accessSecretKey = "secret_access_key"
	)

	config := maps.Clone(opts)
	config[dirMarkersKey] = dirMarkers
	config[listVersionKey] = listVersion
	config[listChunkKey] = listChunk

	// required options
	const (
		chunkSizeKey  = "chunk_size"
		copyCutoffKey = "copy_cutoff"

		defaultChunkSize  = "5MiB"
		defaultCopyCutoff = "5MiB"
	)

	if bucket := config[bucketKey]; bucket == "" {
		return nil, ErrMissingBucket
	}

	internal.SetDefaultValue(config, copyCutoffKey, defaultCopyCutoff)
	internal.SetDefaultValue(config, chunkSizeKey, defaultChunkSize)

	if key != "" {
		config[accessIDKey] = key
	}

	if secret != "" {
		config[accessSecretKey] = secret
	}

	return config, nil
}

func newVFS(name, key, secret string, opts map[string]string) (*vfs.VFS, error) {
	config, err := parseOpts(key, secret, opts)
	if err != nil {
		return nil, err
	}

	root := config[bucketKey]

	fullConfig, err := internal.WithDefaults("s3", config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	s3fs, err := s3.NewFs(context.Background(), name, root, fullConfig)
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
