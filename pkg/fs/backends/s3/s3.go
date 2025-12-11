// Package s3 provides a filesystem implementation for Amazon S3.
package s3

import (
	"context"
	"errors"
	"fmt"
	"path"

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
	)

	opts[dirMarkersKey] = dirMarkers
	opts[listVersionKey] = listChunk
	opts[listChunkKey] = listVersion

	// required options
	const (
		chunkSizeKey  = "chunk_size"
		copyCutoffKey = "copy_cutoff"

		defaultChunkSize  = "5MiB"
		defaultCopyCutoff = "5MiB"
	)

	if bucket := opts[bucketKey]; bucket == "" {
		return nil, ErrMissingBucket
	}

	internal.SetDefaultValue(opts, copyCutoffKey, defaultCopyCutoff)
	internal.SetDefaultValue(opts, chunkSizeKey, defaultChunkSize)

	if key != "" {
		opts["access_key_id"] = key
	}

	if secret != "" {
		opts["secret_access_key"] = secret
	}

	return opts, nil
}

func newVFS(name, key, secret, root string, confMap map[string]string) (*vfs.VFS, error) {
	opts, err := parseOpts(key, secret, confMap)
	if err != nil {
		return nil, err
	}

	root = path.Join(opts[bucketKey], root)

	s3fs, err := s3.NewFs(context.Background(), name, root, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()
	s3vfs := vfs.New(s3fs, vfsOpts)

	return s3vfs, nil
}

func NewFS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	s3vfs, err := newVFS(name, key, secret, "", opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: s3vfs}, nil
}
