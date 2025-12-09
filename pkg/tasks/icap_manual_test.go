//go:build manual_test

package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestManualIcap(t *testing.T) {
	const (
		filename    = "test.txt"
		fileContent = "Hello World"
	)

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{
			Filesize: int64(len(fileContent)),
		},
		Rule: &model.Rule{
			IsSend: true,
		},
		RemoteAgent: &model.RemoteAgent{
			Address: types.Addr("127.0.0.1", 6666),
		},
	}

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		task := &icapTask{}
		params := map[string]string{
			"uploadURL": "127.0.0.1:1344/echo",
			"timeout":   "5h",
		}

		require.NoError(t, task.Run(t.Context(), params, db, logger, transCtx, nil))
	})
}
