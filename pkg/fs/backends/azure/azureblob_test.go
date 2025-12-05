package azure

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestAzureBlob(t *testing.T) {
	t.Parallel()

	opts := map[string]string{
		"env_auth": "true",
	}

	blobVFS, fsErr := newBlobVFS("azblob", "", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, blobVFS)
}
