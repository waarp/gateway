package azure

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestAzureFiles(t *testing.T) {
	t.Parallel()

	const shareName = "gwfiles"

	opts := map[string]string{
		"env_auth":   "true",
		"share_name": shareName,
	}

	filesVFS, fsErr := newFilesVFS("azfiles", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, filesVFS)
}
