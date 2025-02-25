// Package fstest provides filesystem utilities for testing the gateway.
package fstest

import (
	"errors"
	"maps"
	"testing"

	"github.com/rclone/rclone/fs/object"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func MakeDummyBackend(tb testing.TB) string {
	tb.Helper()

	kind := tb.Name()
	fs.Register(kind, func(name, key, secret string, opts map[string]string) (fs.FS, error) {
		return &fs.VFS{VFS: vfs.New(object.MemoryFs, &vfscommon.Options{})}, nil
	})
	tb.Cleanup(func() { fs.Unregister(kind) })

	return kind
}

var ErrInvalidConfig = errors.New("invalid filesystem configuration")

func MakeStaticBackend(tb testing.TB, expName, expKey, expSecret string,
	expOpts map[string]string,
) string {
	tb.Helper()

	kind := tb.Name()
	fs.Register(kind, func(name, key, secret string, opts map[string]string) (fs.FS, error) {
		if name != expName || key != expKey || secret != expSecret ||
			!maps.Equal(opts, expOpts) {
			return nil, ErrInvalidConfig
		}

		return &fs.VFS{VFS: vfs.New(object.MemoryFs, &vfscommon.Options{})}, nil
	})
	tb.Cleanup(func() { fs.Unregister(kind) })

	return kind
}
