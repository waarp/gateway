// Package s3 provides a filesystem implementation for Amazon S3.
package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/rclone/rclone/backend/s3"
	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

//nolint:gochecknoinits //init is used by design
func init() {
	fs.Register("s3", newS3FS)
}

func parseOpts(opts map[string]string) (configmap.Simple, *vfscommon.Options, error) {
	const (
		oldBucketKey = "bucket"
		newBucketKey = "bucket_acl"
	)

	confMap := configmap.Simple(opts)
	if confMap[newBucketKey] == "" && confMap[oldBucketKey] != "" {
		confMap[newBucketKey] = confMap[oldBucketKey]
	}

	vfsOpts, err := parseVFSOpts(opts)

	return confMap, vfsOpts, err
}

func parseVFSOpts(opts map[string]string) (*vfscommon.Options, error) {
	const (
		cacheMaxAgeKey  = "cacheMaxAge"
		cacheMaxSizeKey = "cacheMaxSize"

		readWaitDuration = rfs.Duration(100 * time.Millisecond)
	)

	vfsOpts := &vfscommon.Options{
		FastFingerprint: true,
		CacheMode:       vfscommon.CacheModeOff,
		ReadWait:        readWaitDuration,
	}

	if age, ok := opts[cacheMaxAgeKey]; ok {
		if err := vfsOpts.CacheMaxAge.Set(age); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", cacheMaxAgeKey, err)
		}
	}

	if size, ok := opts[cacheMaxSizeKey]; ok {
		if err := vfsOpts.CacheMaxSize.Set(size); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", cacheMaxSizeKey, err)
		}
	}

	return vfsOpts, nil
}

func newS3FS(name, key, secret string, opts map[string]string) (fs.FS, error) {
	confMap, vfsOpts, err := parseOpts(opts)
	if err != nil {
		return nil, err
	}

	confMap["access_key_id"] = key
	confMap["secret_access_key"] = secret

	s3fs, err := s3.NewFs(context.Background(), name, "", confMap)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate s3 filesystem: %w", err)
	}

	s3vfs := vfs.New(s3fs, vfsOpts)

	return &fs.VFS{VFS: s3vfs}, nil
}
