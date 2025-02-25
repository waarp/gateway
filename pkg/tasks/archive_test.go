package tasks

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestArchiveZip(t *testing.T) {
	testArchive(t, ".zip")
}

func TestArchiveTar(t *testing.T) {
	testArchive(t, ".tar")
}

func TestArchiveTarGzip(t *testing.T) {
	testArchive(t, ".tar.gz")
}

func TestArchiveTarBzip2(t *testing.T) {
	testArchive(t, ".tar.bz2")
}

func TestArchiveTarXz(t *testing.T) {
	testArchive(t, ".tar.xz")
}

func TestArchiveTarZstd(t *testing.T) {
	testArchive(t, ".tar.zstd")
}

func TestArchiveTarZlib(t *testing.T) {
	testArchive(t, ".tar.zlib")
}

func testArchive(t *testing.T, extension string) {
	t.Helper()

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	root := t.TempDir()
	dir := fs.JoinPath(root, "dir")
	subDir := fs.JoinPath(dir, "subdir")
	require.NoError(t, fs.MkdirAll(subDir))

	fileA := fs.JoinPath(root, "fileA")
	fileB := fs.JoinPath(root, "fileB")

	file1 := fs.JoinPath(dir, "file1")
	file2 := fs.JoinPath(dir, "file2")
	file3 := fs.JoinPath(subDir, "file3")

	require.NoError(t, fs.WriteFullFile(fileA, []byte("contentA")))
	require.NoError(t, fs.WriteFullFile(fileB, []byte("contentB")))
	require.NoError(t, fs.WriteFullFile(file1, []byte("content1")))
	require.NoError(t, fs.WriteFullFile(file2, []byte("content2")))
	require.NoError(t, fs.WriteFullFile(file3, []byte("content3")))

	archivePath := fs.JoinPath(root, "output"+extension)

	pattern := path.Join(root, "file[A-Z]")
	files := []string{pattern, dir}

	archParam := map[string]string{
		"files":      strings.Join(files, ", "),
		"outputPath": archivePath,
	}

	extrParams := map[string]string{
		"outputDir": root,
	}

	transCtxArch := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: dir},
	}

	transCtxExtr := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: archivePath},
	}

	archive := &archiveTask{}
	extract := &extractTask{}
	ctx := context.Background()

	require.NoError(t, archive.Run(ctx, archParam, db, logger, transCtxArch))
	assert.FileExists(t, archivePath)

	require.NoError(t, fs.RemoveAll(fileA))
	require.NoError(t, fs.RemoveAll(fileB))
	require.NoError(t, fs.RemoveAll(dir))

	require.NoError(t, extract.Run(ctx, extrParams, db, logger, transCtxExtr))

	assert.FileExists(t, fileA)
	assert.FileExists(t, fileB)
	assert.DirExists(t, dir)
	assert.FileExists(t, file1)
	assert.FileExists(t, file2)
	assert.DirExists(t, subDir)
	assert.FileExists(t, file3)

	dataA, err := fs.ReadFullFile(fileA)
	require.NoError(t, err)

	dataB, err := fs.ReadFullFile(fileB)
	require.NoError(t, err)

	data1, err := fs.ReadFullFile(file1)
	require.NoError(t, err)

	data2, err := fs.ReadFullFile(file2)
	require.NoError(t, err)

	data3, err := fs.ReadFullFile(file3)
	require.NoError(t, err)

	assert.Equal(t, "contentA", string(dataA))
	assert.Equal(t, "contentB", string(dataB))
	assert.Equal(t, "content1", string(data1))
	assert.Equal(t, "content2", string(data2))
	assert.Equal(t, "content3", string(data3))
}
