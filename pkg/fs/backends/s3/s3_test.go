package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestS3(t *testing.T) {
	const bucket = "waarp-gateway-tests"

	backtest.EnvOrSkip(t, "AWS_ACCESS_KEY_ID")
	backtest.EnvOrSkip(t, "AWS_SECRET_ACCESS_KEY")

	opts := map[string]string{
		"env_auth": "true",
		"bucket":   bucket,
	}

	s3VFS, fsErr := newVFS("s3", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, s3VFS)
}

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts := map[string]string{"bucket": "name"}
	_, err := parseOpts("account", "password", opts)
	require.NoError(t, err)

	assert.NotContains(t, opts, "access_key_id")
	assert.NotContains(t, opts, "secret_access_key")
}
