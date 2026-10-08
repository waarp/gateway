package tasks

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestChmodTaskRun(t *testing.T) {
	t.Parallel()

	logger := logtest.GetTestLogger(t)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		path := filepath.Join(root, "testfile.txt")

		var sourcePerms, destPerms os.FileMode = 0o755, 0o644
		if runtime.GOOS == "windows" {
			sourcePerms, destPerms = 0o666, 0o444
		}
		require.NoError(t, os.WriteFile(path, []byte("hello world"), sourcePerms))

		args := map[string]string{"perms": strconv.FormatUint(uint64(destPerms), 8)}
		transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path}}
		task := &chmodTask{}

		require.NoError(t, task.Run(t.Context(), args, nil, logger, transCtx, nil))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, destPerms, info.Mode().Perm())
	})

	t.Run("Illegal bits", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		path := filepath.Join(root, "testfile.txt")

		var sourcePerms, destPerms os.FileMode = 0o755, 0o2644
		if runtime.GOOS == "windows" {
			sourcePerms, destPerms = 0o666, 0o2444
		}
		require.NoError(t, os.WriteFile(path, []byte("hello world"), sourcePerms))

		args := map[string]string{"perms": strconv.FormatUint(uint64(destPerms), 8)}
		transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path}}
		task := &chmodTask{}

		var bitErr ChmodBitsError
		require.ErrorAs(t, task.Run(t.Context(), args, nil, logger, transCtx, nil), &bitErr)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, sourcePerms, info.Mode().Perm())
	})
}
