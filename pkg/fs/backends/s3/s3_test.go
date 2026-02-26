package s3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestS3(t *testing.T) {
	const bucket = "waarp-gateway-tests"

	opts := map[string]string{
		"env_auth": "true",
		"bucket":   bucket,
	}

	s3VFS, fsErr := newVFS("s3", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, s3VFS)
}
