// Package fstest provides an in-memory filesystem implementation for testing.
package fstest

import (
	"testing"

	"github.com/hack-pad/hackpadfs/mem"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
)

const MemScheme = "memory"

func Init(tb testing.TB, scheme string) *mem.FS {
	tb.Helper()

	memFS, err := mem.NewFS()
	require.NoError(tb, err, "failed to create memfs")

	filesystems.TestFileSystems[scheme] = memFS

	tb.Cleanup(func() {
		delete(filesystems.TestFileSystems, scheme)
	})

	return memFS
}

func InitMemFS(c convey.C) *mem.FS {
	memFS, err := mem.NewFS()
	c.So(err, convey.ShouldBeNil)

	filesystems.TestFileSystems[MemScheme] = memFS

	c.Reset(func() {
		// c.So(hfs.RemoveAll(memFS, "."), convey.ShouldBeNil)
		delete(filesystems.TestFileSystems, MemScheme)
	})

	return memFS
}
