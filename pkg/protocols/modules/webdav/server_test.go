package webdav_test

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestServerPropfind(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)
	client := makeClient(ctx)

	// Setup data
	dirPath := filepath.Join(ctx.Root, ctx.RulePull.LocalDir)
	require.NoError(t, os.MkdirAll(dirPath, 0o700))
	filePath := filepath.Join(dirPath, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o600))

	// Do
	dirURL := ctx.RulePull.Path
	fileURL := path.Join(dirURL, "test.txt")

	require.NoError(t, client.Connect())
	_, err := client.Stat(dirURL)
	require.NoError(t, err)
	_, err = client.Stat(fileURL)
	require.NoError(t, err)
}

func TestServerUpload(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)
	client := makeClient(ctx)

	// Setup data
	srcBuf := makeBuf(t)
	destFilename := "upload.txt"
	destPath := path.Join(ctx.RulePush.Path, destFilename)
	dstFilePath := filepath.Join(ctx.Root, ctx.RulePush.LocalDir, destFilename)

	// Do transfer
	require.NoError(t, client.Connect())
	require.NoError(t, client.WriteStreamWithLength(destPath, bytes.NewReader(srcBuf),
		int64(len(srcBuf)), 0o600))
	ctx.WaitEnd(t)

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", destFilename).Run())
	assert.Equal(t, ctx.Server.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePush.Name, hist.Rule)
	assert.Equal(t, ctx.RulePush.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, destFilename, hist.DestFilename)
	assert.Equal(t, int64(buffSize), hist.Filesize)
	assert.Equal(t, filepath.ToSlash(dstFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	content, err := os.ReadFile(dstFilePath)
	require.NoError(t, err)
	assert.Equal(t, srcBuf, content)
}

func TestServerDownload(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)
	client := makeClient(ctx)

	// Setup data
	srcBuf := makeBuf(t)
	srcFilename := t.Name() + ".src"
	srcPath := path.Join(ctx.RulePull.Path, srcFilename)
	srcFilePath := filepath.Join(ctx.Root, ctx.RulePull.LocalDir, srcFilename)
	require.NoError(t, os.WriteFile(srcFilePath, srcBuf, 0o600))

	// Do transfer
	require.NoError(t, client.Connect())
	destBuf, err := client.Read(srcPath)
	require.NoError(t, err)
	ctx.WaitEnd(t)

	// Check history
	var hist model.HistoryEntry
	require.NoError(t, ctx.DB.Get(&hist, "src_filename=?", srcFilename).Run())
	assert.Equal(t, ctx.Server.Name, hist.Agent)
	assert.Equal(t, ctx.Account.Login, hist.Account)
	assert.Equal(t, ctx.RulePull.Name, hist.Rule)
	assert.Equal(t, ctx.RulePull.IsSend, hist.IsSend)
	assert.Equal(t, types.StatusDone, hist.Status)
	assert.Equal(t, filepath.ToSlash(srcFilePath), hist.LocalPath)
	assert.Zero(t, hist.ErrCode)
	assert.Zero(t, hist.ErrDetails)

	// Check file
	assert.Equal(t, srcBuf, destBuf)
}

func TestServerPreTaskError(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)
	expErrCode, expErrMsg := ctx.AddPushPreTaskError(t)
	client := makeClient(ctx)

	// Setup data
	filename := "upload.txt"
	destPath := path.Join(ctx.RulePush.Path, filename)

	// Do transfer
	require.NoError(t, client.Connect())
	require.Error(t, client.WriteStreamWithLength(destPath, &bytes.Reader{}, 0, 0o600))
	ctx.WaitEnd(t)

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
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)
	expErrCode, expErrMsg := ctx.AddPushPostTaskError(t)
	client := makeClient(ctx)

	// Setup data
	filename := "upload.txt"
	destPath := path.Join(ctx.RulePush.Path, filename)

	// Do transfer
	require.NoError(t, client.Connect())
	require.Error(t, client.WriteStreamWithLength(destPath, &bytes.Reader{}, 0, 0o600))
	ctx.WaitEnd(t)

	// Check history
	var hist model.Transfer
	require.NoError(t, ctx.DB.Get(&hist, "dest_filename=?", filename).Run())
	assert.Equal(t, types.StatusError, hist.Status)
	assert.Equal(t, types.StepPostTasks, hist.Step)
	assert.Equal(t, expErrCode, hist.ErrCode)
	assert.Equal(t, expErrMsg, hist.ErrDetails)
}

func TestServerTLS(t *testing.T) {
	t.Parallel()

	// Setup DB
	ctx := gwtesting.NewTestServerCtx(t, webdav.WebdavTLS, nil)
	ctx.AddPassword(t, password)
	ctx.AddCert(t, gwtesting.LocalhostCertPEM, gwtesting.LocalhostKeyPEM)
	client := makeClient(ctx)

	// Do transfer
	require.NoError(t, client.Connect())
}
