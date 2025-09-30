// Package s3 provides a filesystem implementation for Amazon S3.
package s3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/rclone/rclone/backend/s3"
	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

var ErrMissingBucket = errors.New("no S3 bucket specified")

func parseOpts(opts map[string]string) (configmap.Simple, *vfscommon.Options) {
	const (
		envAuthKey = "env_auth"
		envAuth    = "true"

		chunkSizeKey = "chunk_size"
		chunkSize    = "5MiB"

		copyCutoffKey = "copy_cutoff"
		copyCutoff    = "5MiB"

		dirMarkersKey = "directory_markers"
		dirMarkers    = "true"

		regionKey     = "region"
		regionEnvVar  = "AWS_REGION"
		regionEnvVar2 = "AWS_DEFAULT_REGION"

		listChunkKey   = "list_chunk"
		listChunk      = "1000"
		listVersionKey = "list_version"
		listVersion    = "2"
	)

	confMap := configmap.Simple(opts)

	if confMap[regionKey] == "" {
		confMap[regionKey] = os.Getenv(regionEnvVar)
	}

	if confMap[regionKey] == "" {
		confMap[regionKey] = os.Getenv(regionEnvVar2)
	}

	confMap[envAuthKey] = envAuth
	confMap[chunkSizeKey] = chunkSize
	confMap[copyCutoffKey] = copyCutoff
	confMap[dirMarkersKey] = dirMarkers
	confMap[listChunkKey] = listChunk
	confMap[listVersionKey] = listVersion

	vfsOpts := parseVFSOpts()

	return confMap, vfsOpts
}

func parseVFSOpts() *vfscommon.Options {
	const (
		readWaitDuration = rfs.Duration(100 * time.Millisecond)
	)

	vfsOpts := &vfscommon.Options{
		FastFingerprint: true,
		CacheMode:       vfscommon.CacheModeOff,
		ReadWait:        readWaitDuration,
	}

	return vfsOpts
}

func NewFS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	s3vfs, err := newS3FSWithRoot(name, key, secret, "", opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: s3vfs}, nil
}

func newS3FSWithRoot(name, key, secret, root string, opts map[string]string) (*vfs.VFS, error) {
	confMap, vfsOpts := parseOpts(opts)

	if key != "" {
		confMap["access_key_id"] = key
	}

	if secret != "" {
		confMap["secret_access_key"] = secret
	}

	if bucket := confMap["bucket"]; bucket == "" {
		return nil, ErrMissingBucket
	} else {
		root = path.Join(bucket, root)
	}

	s3fs, err := s3.NewFs(context.Background(), name, root, confMap)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	s3vfs := vfs.New(s3fs, vfsOpts)

	return s3vfs, nil
}
