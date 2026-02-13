//go:build manual_test

package webdav_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestManual(t *testing.T) {
	ctx := gwtesting.NewTestServerCtx(t, webdav.Webdav, nil)
	ctx.AddPassword(t, password)

	testFilePath := filepath.Join(ctx.Root, ctx.RulePull.LocalDir, "pull.txt")
	require.NoError(t, os.WriteFile(testFilePath, []byte("hello world"), 0o600))

	<-t.Context().Done()
}
