package backtest

import (
	"testing"

	"github.com/rclone/rclone/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func TestVFS(tb testing.TB, rvfs *vfs.VFS) {
	tb.Helper()

	const (
		dirName    = "foo"
		filename   = "bar.txt"
		filepath   = dirName + "/" + filename
		expContent = "hello world"
	)

	require.NoError(tb, rvfs.Mkdir(dirName, 0o700))

	file, opErr := rvfs.OpenFile(filepath, fs.FlagReadWrite|fs.FlagCreate|fs.FlagTruncate, 0o600)
	require.NoError(tb, opErr)

	tb.Cleanup(func() {
		require.NoError(tb, file.Close())

		cont, rErr := rvfs.ReadFile(filepath)
		require.NoError(tb, rErr)
		assert.Equal(tb, expContent, string(cont))

		require.NoError(tb, rvfs.Remove(filepath))
		_, fErr := rvfs.Stat(filepath)
		assert.ErrorIs(tb, fErr, vfs.ENOENT)

		require.NoError(tb, rvfs.Remove(dirName))
		_, dErr := rvfs.Stat(filepath)
		assert.ErrorIs(tb, dErr, vfs.ENOENT)
	})

	_, rErr := file.WriteString(expContent)
	require.NoError(tb, rErr)
}
