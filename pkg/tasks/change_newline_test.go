package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestChangeNewline(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	const (
		fromSeparator = "\012"
		toSeparator   = "\x0D\x0A"
		origContent   = "Hello World" + fromSeparator +
			"This is a multiline text file" + fromSeparator +
			"To test the CHNEWLINE task." + fromSeparator +
			"Bye bye."
	)

	expectedContent := strings.ReplaceAll(origContent, fromSeparator, toSeparator)

	filePath := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte(origContent), 0o600))

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: filepath.ToSlash(filePath)},
	}

	args := map[string]string{
		"from": fromSeparator,
		"to":   toSeparator,
	}

	task := chNewlineTask{}
	require.NoError(t, task.Run(t.Context(), args, db, logger, transCtx, nil))

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}
