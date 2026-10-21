package gcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestGCS(t *testing.T) {
	t.Parallel()

	const (
		bucket    = "waarp_gateway_test"
		projectNb = "1048684374782"
	)

	backtest.EnvOrSkip(t, "GOOGLE_APPLICATION_CREDENTIALS")

	opts := map[string]string{
		"bucket":             bucket,
		"env_auth":           "true",
		"project_number":     projectNb,
		"bucket_policy_only": "true",
	}

	gcsVFS, fsErr := newVFS("gcs", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, gcsVFS)
}

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts := map[string]string{"bucket": "name"}
	conf, _, err := parseOpts("file", "creds", opts)
	require.NoError(t, err)

	assert.NotContains(t, opts, "service_account_credentials")
	assert.NotContains(t, opts, "service_account_file")

	file, _ := conf.Get("service_account_file")
	assert.Equal(t, "file", file)

	creds, _ := conf.Get("service_account_credentials")
	assert.Equal(t, "creds", creds)
}
