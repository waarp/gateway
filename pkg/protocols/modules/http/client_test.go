package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestClientError(t *testing.T) {
	runTest := func(tb testing.TB, isSend bool, status int, msg string) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, msg, status)
		}))
		defer server.Close()

		const filename = "test.txt"
		ctx := gwtesting.NewTestClientCtx(tb, HTTP, server.Listener.Addr().String(), nil, nil)

		// Setup Data
		if isSend {
			srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
			require.NoError(t, os.WriteFile(srcFilePath, []byte("hello world"), 0o600))
		}

		if isSend {
			assert.Error(tb, ctx.RunUpload(tb, filename))
		} else {
			assert.Error(tb, ctx.RunDownload(tb, filename))
		}

		var check model.Transfer
		require.NoError(tb, ctx.DB.Get(&check, "").Run())
		assert.Equal(tb, statusToErrCode(status), check.ErrCode)
		assert.Contains(tb, check.ErrDetails, msg)
	}

	runTests := func(t *testing.T, isSend bool) {
		t.Helper()

		for _, testCase := range []struct {
			status int
			msg    string
		}{
			{http.StatusUnauthorized, "authentication refused"},
			{http.StatusForbidden, "permission denied"},
			{http.StatusNotFound, "file not found"},
			{http.StatusInternalServerError, "internal server error"},
		} {
			t.Run(utils.FormatInt(testCase.status), func(t *testing.T) {
				runTest(t, isSend, testCase.status, testCase.msg)
			})
		}
	}

	t.Run("Push", func(t *testing.T) {
		runTests(t, true)
	})

	t.Run("Pull", func(t *testing.T) {
		runTests(t, false)
	})
}
