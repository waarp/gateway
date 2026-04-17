package as2_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2/internal/as2test"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPlain(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, false, nil, nil)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2, server.Addr, nil, nil)
	ctx.AddPassword(t, as2test.Password)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	require.NoError(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, ctx.Client.Name, hist.Client)
	assert.Equal(t, ctx.Partner.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(srcFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content := server.ReadFile(t, filename)
	assert.Equal(t, srcBuf, content)
}

func TestClientTLS(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, true, nil, &gwtesting.ServerCert)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2TLS, server.Addr, nil, nil)
	ctx.AddPassword(t, as2test.Password)
	ctx.AddCert(t, gwtesting.ServerCertPEM)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	require.NoError(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, ctx.Client.Name, hist.Client)
	assert.Equal(t, ctx.Partner.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(srcFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content := server.ReadFile(t, filename)
	assert.Equal(t, srcBuf, content)
}

func TestClientPreTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, false, nil, nil)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2, server.Addr, nil, nil)
	ctx.AddPassword(t, as2test.Password)
	expErrCode, expErrMsg := ctx.AddPushPreTaskError(t)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	assert.Error(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPreTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}

func TestClientPostTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, false, nil, nil)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2, server.Addr, nil, nil)
	ctx.AddPassword(t, as2test.Password)
	expErrCode, expErrMsg := ctx.AddPushPostTaskError(t)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	assert.Error(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPostTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}

func TestClientSignEncrypt(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, false, gwtesting.ClientCert.Leaf, &gwtesting.ServerCert)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2, server.Addr, nil, map[string]any{
		"signatureAlgorithm":  as2.SignAlgoSHA256,
		"encryptionAlgorithm": as2.EncryptAlgoAES256CBC,
	})
	ctx.AddPassword(t, as2test.Password)
	ctx.AddCert(t, gwtesting.ServerCertPEM)
	ctx.AddClientCert(t, gwtesting.ClientCertPEM, gwtesting.ClientKeyPEM)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	require.NoError(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, ctx.Client.Name, hist.Client)
	assert.Equal(t, ctx.Partner.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(srcFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content := server.ReadFile(t, filename)
	assert.Equal(t, len(srcBuf), len(content))
	assert.Equal(t, srcBuf, content)
}

func TestClientAsyncMDN(t *testing.T) {
	t.Parallel()

	// Setup DB
	server := as2test.MakeServer(t, false, nil, nil)
	ctx := gwtesting.NewTestClientCtx(t, as2.AS2, server.Addr, nil, map[string]any{
		"asyncMDNAddress": gwtesting.GetLocalAddr(t).String(),
		"handleAsyncMDN":  true,
	})
	ctx.AddPassword(t, as2test.Password)

	// Setup Data
	const filename = "upload.txt"
	srcBuf := makeBuf(t)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, filename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	require.NoError(t, ctx.RunUpload(t, filename))

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, ctx.Client.Name, hist.Client)
	assert.Equal(t, ctx.Partner.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(srcFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content := server.ReadFile(t, filename)
	assert.Equal(t, srcBuf, content)
}
