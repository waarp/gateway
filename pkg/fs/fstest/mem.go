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

const MemBackend = "memory"

func Init(tb testing.TB, scheme string) *mem.FS {
	tb.Helper()

	memFS, err := mem.NewFS()
	require.NoError(tb, err, "failed to create memfs")

	filesystems.TestFileSystems.Store(scheme, memFS)

	tb.Cleanup(func() {
		require.NoError(tb, hfs.RemoveAll(memFS, "."))
		filesystems.TestFileSystems.Delete(scheme)
	})

	return memFS
}

func InitMemFS(c convey.C) fs.FS {
	memFS, err := mem.NewFS()
	c.So(err, convey.ShouldBeNil)

	_, ok := filesystems.TestFileSystems.Load(MemBackend)
	c.So(ok, convey.ShouldBeFalse)

	filesystems.TestFileSystems.Store(MemBackend, memFS)

	c.Reset(func() {
		c.So(hfs.RemoveAll(memFS, "."), convey.ShouldBeNil)
		filesystems.TestFileSystems.Delete(MemBackend)
	})

	return memFS
}
