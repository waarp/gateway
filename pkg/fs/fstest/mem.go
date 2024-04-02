// Package fstest provides an in-memory filesystem implementation for testing.
package fstest

import (
	"testing"

	hfs "github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
)

const MemScheme = "memory"

func Init(tb testing.TB, scheme string) *mem.FS {
	tb.Helper()

	memFS, err := mem.NewFS()
	require.NoError(tb, err, "failed to create memfs")

	filesystems.TestFileSystems[scheme] = memFS

	tb.Cleanup(func() {
		require.NoError(tb, hfs.RemoveAll(memFS, "."))
		delete(filesystems.TestFileSystems, scheme)
	})

	return memFS
}

func InitMemFS(c convey.C) fs.FS {
	memFS, err := mem.NewFS()
	c.So(err, convey.ShouldBeNil)

	c.So(filesystems.TestFileSystems, convey.ShouldNotContainKey, MemScheme)
	filesystems.TestFileSystems[MemScheme] = memFS

	c.Reset(func() {
		c.So(hfs.RemoveAll(memFS, "."), convey.ShouldBeNil)
		delete(filesystems.TestFileSystems, MemScheme)
	})

	return memFS
}
