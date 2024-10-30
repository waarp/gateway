package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestTranscode(t *testing.T) {
	root := t.TempDir()
	logger := testhelpers.GetTestLogger(t)

	// just a bunch of non-ASCII characters in a UTF-8 string
	const testString = "âàäçéêèîïôöûùüÿ"
	// this is the same string in Windows 1252 encoding
	testBytes := []byte{226, 224, 228, 231, 233, 234, 232, 238, 239, 244, 246, 251, 249, 252, 255}
	require.NotEqual(t, testString, string(testBytes)) // sanity check

	filepath := fs.JoinPath(root, "test.txt")
	require.NoError(t, fs.WriteFullFile(filepath, testBytes))

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: filepath},
	}

	params := map[string]string{
		"fromCharset": "Windows 1252",
		"toCharset":   "UTF-8",
	}

	task := &transcodeTask{}
	require.NoError(t, task.Run(context.Background(), params, nil, logger, transCtx))

	content, err := fs.ReadFullFile(transCtx.Transfer.LocalPath)
	require.NoError(t, err)

	assert.Equal(t, testString, string(content))
}
