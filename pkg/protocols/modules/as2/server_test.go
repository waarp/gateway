package as2_test

import (
	"net/http"
	"os"
	"path"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	as2lib "code.waarp.fr/lib/as2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2/internal/as2test"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestServerPlain(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, as2.AS2, map[string]any{
		"maxFileSize": as2test.MaxBodySize,
	})
	client := as2test.MakeClient(t, false, ctx.Server.Address.String(), nil, nil)

	// Setup Data
	const filename = "upload.txt"
	dstPath := path.Join(ctx.RulePush.Path, filename)
	dstFilePath := fs.JoinPath(ctx.Root, ctx.RulePush.LocalDir, filename)

	// Do transfer
	res := client.Run(t, dstPath)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, as2lib.MDNStatusSuccess, res.MDN.Status)
	t.Logf("MDN: %s", res.MDN.HumanText)

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", filename).Run())
	assert.Equal(t, ctx.Server.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, dstFilePath, hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content, err := os.ReadFile(dstFilePath)
	require.NoError(t, err)
	assert.Equal(t, client.FileContent, content)
}

func TestServerTLS(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, as2.AS2TLS, map[string]any{
		"maxFileSize": as2test.MaxBodySize,
		//"mdnSignatureAlgorithm": as2.SignAlgoSHA256,
	})
	client := as2test.MakeClient(t, true, ctx.Server.Address.String(), nil,
		gwtesting.ServerCert.Leaf)
	ctx.AddCert(t, gwtesting.ServerCertPEM, gwtesting.ServerKeyPEM)
	ctx.RestartServer(t)

	// Setup Data
	const filename = "upload.txt"
	dstPath := path.Join(ctx.RulePush.Path, filename)
	dstFilePath := fs.JoinPath(ctx.Root, ctx.RulePush.LocalDir, filename)

	// Do transfer
	res := client.Run(t, dstPath)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, as2lib.MDNStatusSuccess, res.MDN.Status)
	t.Logf("MDN: %s", res.MDN.HumanText)

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", filename).Run())
	assert.Equal(t, ctx.Server.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, dstFilePath, hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content, err := os.ReadFile(dstFilePath)
	require.NoError(t, err)
	assert.Equal(t, client.FileContent, content)
}

func TestServerPreTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, as2.AS2, map[string]any{
		"maxFileSize": as2test.MaxBodySize,
	})
	client := as2test.MakeClient(t, false, ctx.Server.Address.String(), nil, nil)
	expErrCode, expErrMsg := ctx.AddPushPreTaskError(t)

	// Setup Data
	const filename = "upload.txt"
	dstPath := path.Join(ctx.RulePush.Path, filename)

	// Do transfer
	res := client.Run(t, dstPath)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, as2lib.MDNStatusError, res.MDN.Status)
	t.Logf("MDN: %s", res.MDN.HumanText)

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPreTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}

func TestServerPostTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, as2.AS2, map[string]any{
		"maxFileSize": as2test.MaxBodySize,
	})
	client := as2test.MakeClient(t, false, ctx.Server.Address.String(), nil, nil)
	expErrCode, expErrMsg := ctx.AddPushPostTaskError(t)

	// Setup Data
	const filename = "upload.txt"
	dstPath := path.Join(ctx.RulePush.Path, filename)

	// Do transfer
	res := client.Run(t, dstPath)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, as2lib.MDNStatusError, res.MDN.Status)
	t.Logf("MDN: %s", res.MDN.HumanText)

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPostTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}
