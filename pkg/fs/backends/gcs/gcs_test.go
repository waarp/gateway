package gcs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestGCS(t *testing.T) {
	const (
		bucket    = "waarp_gateway_test"
		projectNb = "1048684374782"
	)

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
