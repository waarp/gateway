package webdav_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientUpload(t *testing.T) {
	t.Parallel()

	// Setup DB
	serv := makeServer(t)
	ctx := gwtesting.NewTestClientCtx(t, webdav.Webdav, serv.addr, nil, nil)
	ctx.AddPassword(t, password)

	// Setup Data
	srcBuf := makeBuf(t)
	filename := "upload.txt"
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
	content := serv.readFile(t, filename)
	assert.Equal(t, srcBuf, content)
}

func TestClientDownload(t *testing.T) {
	t.Parallel()

	// Setup DB
	serv := makeServer(t)
	ctx := gwtesting.NewTestClientCtx(t, webdav.Webdav, serv.addr, nil, nil)
	ctx.AddPassword(t, password)

	// Setup Data
	srcBuf := makeBuf(t)
	filename := "upload.txt"
	serv.writeFile(t, filename, srcBuf)

	// Do transfer
	require.NoError(t, ctx.RunDownload(t, filename))

	// Check history
	dstFilepath := filepath.Join(ctx.Root, ctx.RulePull.LocalDir, filename)

	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, ctx.Client.Name, hist.Client)
	assert.Equal(t, ctx.Partner.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePull.Name, hist.Rule)
	assert.Equal(t, ctx.RulePull.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(dstFilepath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content, err := os.ReadFile(dstFilepath)
	require.NoError(t, err)
	assert.Equal(t, srcBuf, content)
}

func TestClientPreTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	serv := makeServer(t)
	ctx := gwtesting.NewTestClientCtx(t, webdav.Webdav, serv.addr, nil, nil)
	ctx.AddPassword(t, password)
	expErrCode, expErrMsg := ctx.AddPullPreTaskError(t)

	// Setup Data
	filename := "upload.txt"
	serv.writeFile(t, filename, []byte{})

	// Do transfer
	assert.Error(t, ctx.RunDownload(t, filename))

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
	serv := makeServer(t)
	ctx := gwtesting.NewTestClientCtx(t, webdav.Webdav, serv.addr, nil, nil)
	ctx.AddPassword(t, password)
	expErrCode, expErrMsg := ctx.AddPullPostTaskError(t)

	// Setup Data
	filename := "upload.txt"
	serv.writeFile(t, filename, []byte{})

	// Do transfer
	assert.Error(t, ctx.RunDownload(t, filename))

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPostTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}

func TestClientTLS(t *testing.T) {
	t.Parallel()

	// Setup DB
	serv := makeServer(t)
	ctx := gwtesting.NewTestClientCtx(t, webdav.WebdavTLS, serv.addr, nil, nil)
	ctx.AddPassword(t, password)
	ctx.AddCert(t, gwtesting.LocalhostCertPEM)

	// Setup Data
	filename := "upload.txt"
	serv.writeFile(t, filename, []byte{})

	// Do transfer
	assert.Error(t, ctx.RunDownload(t, filename))
}
