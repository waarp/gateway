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

	backtest.EnvOrSkip(t, "AZURE_STORAGE_ACCOUNT_NAME")
	backtest.EnvOrSkip(t, "AZURE_TENANT_ID")
	backtest.EnvOrSkip(t, "AZURE_CLIENT_ID")
	backtest.EnvOrSkip(t, "AZURE_CLIENT_SECRET")

	opts := map[string]string{
		"env_auth":   "true",
		"share_name": shareName,
	}

	filesVFS, fsErr := newFilesVFS("azfiles", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, filesVFS)
}
