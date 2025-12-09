package tasks

import (
	"context"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRemoteDeleter struct {
	Err       error
	path      string
	recursive bool
	deadline  time.Time
}

func (t *testRemoteDeleter) Delete(ctx context.Context, path string, recursive bool) error {
	t.path = path
	t.recursive = recursive
	t.deadline, _ = ctx.Deadline()

	return t.Err
}

func TestRemoteDelete(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	mkTask := model.ValidTasks[RemoteDelete]
	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{RemotePath: "path/to/file"},
	}

	t.Run("Default params", func(t *testing.T) {
		t.Parallel()

		args := map[string]string{}
		remote := &testRemoteDeleter{}
		require.NoError(t, mkTask().Run(t.Context(), args, db, logger, transCtx, remote))

		assert.Equal(t, transCtx.Transfer.RemotePath, remote.path)
		assert.False(t, remote.recursive)
	})

	t.Run("With set params", func(t *testing.T) {
		t.Parallel()

		timeout := time.Hour
		args := map[string]string{
			"path":      "path/to/dir",
			"recursive": "true",
			"timeout":   timeout.String(),
		}
		remote := &testRemoteDeleter{}
		require.NoError(t, mkTask().Run(t.Context(), args, db, logger, transCtx, remote))

		assert.Equal(t, args["path"], remote.path)
		assert.True(t, remote.recursive)
		assert.WithinDuration(t, time.Now().Add(timeout), remote.deadline, time.Second)
	})

	t.Run("With error", func(t *testing.T) {
		t.Parallel()

		args := map[string]string{}
		remote := &testRemoteDeleter{Err: assert.AnError}
		require.ErrorIs(t, mkTask().Run(t.Context(), args, db, logger, transCtx, remote), assert.AnError)
	})
}
