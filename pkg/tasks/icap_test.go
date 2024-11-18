package tasks

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestIcapTaskReqModRun(t *testing.T) {
	const (
		filename    = "test.file"
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

	expectedRequest := &testIcapReqModRequest{
		icapHeaders: nil,
		payload: testHTTPRequest{
			url:    "http://" + transCtx.RemoteAgent.Address.String(),
			method: http.MethodPost,
			data:   []byte(fileContent),
		},
	}

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeReqModServer(t, expectedRequest, http.StatusNoContent)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL": icapAddr,
			"timeout":   "5h",
		}

		require.NoError(t, task.Run(ctx, params, db, logger, transCtx))
	})

	t.Run("Error with delete", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeReqModServer(t, expectedRequest, http.StatusBadRequest)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL": icapAddr,
			"timeout":   "5s",
			"onError":   icapOnErrorDelete,
		}

		require.Error(t, task.Run(ctx, params, db, logger, transCtx))
		require.NoFileExists(t, filepath)
	})

	t.Run("Error with move", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeReqModServer(t, expectedRequest, http.StatusBadRequest)
		errorPath := fs.JoinPath(root, "quarantine", filename)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL":       icapAddr,
			"timeout":         "5s",
			"onError":         icapOnErrorMove,
			"onErrorMovePath": errorPath,
		}

		require.Error(t, task.Run(ctx, params, db, logger, transCtx))
		require.NoFileExists(t, filepath)
		require.FileExists(t, errorPath)
	})
}

func TestIcapTaskRespModRun(t *testing.T) {
	const (
		filename    = "test.file"
		fileContent = "Hello World"
	)

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{
			Filesize: int64(len(fileContent)),
		},
		Rule: &model.Rule{
			IsSend: false,
		},
		RemoteAgent: &model.RemoteAgent{
			Address: types.Addr("127.0.0.1", 6666),
		},
	}

	expectedRequest := &testIcapRespModRequest{
		icapHeaders: nil,
		payload: testHTTPResponse{
			code: http.StatusOK,
			data: []byte(fileContent),
		},
	}

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeRespModServer(t, expectedRequest, http.StatusNoContent)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL": icapAddr,
			"timeout":   "5s",
		}

		require.NoError(t, task.Run(ctx, params, db, logger, transCtx))
	})

	t.Run("Error with delete", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeRespModServer(t, expectedRequest, http.StatusBadRequest)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL": icapAddr,
			"timeout":   "5s",
			"onError":   icapOnErrorDelete,
		}

		require.Error(t, task.Run(ctx, params, db, logger, transCtx))
		require.NoFileExists(t, filepath)
	})

	t.Run("Error with move", func(t *testing.T) {
		root := t.TempDir()
		filepath := fs.JoinPath(root, filename)
		require.NoError(t, fs.WriteFullFile(filepath, []byte(fileContent)))
		transCtx.Transfer.LocalPath = filepath

		icapAddr := makeRespModServer(t, expectedRequest, http.StatusBadRequest)
		errorPath := fs.JoinPath(root, "quarantine", filename)

		task := &icapTask{}
		params := map[string]string{
			"uploadURL":       icapAddr,
			"timeout":         "5s",
			"onError":         icapOnErrorMove,
			"onErrorMovePath": errorPath,
		}

		require.Error(t, task.Run(ctx, params, db, logger, transCtx))
		require.NoFileExists(t, filepath)
		require.FileExists(t, errorPath)
	})
}
