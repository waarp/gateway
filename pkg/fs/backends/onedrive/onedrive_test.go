package onedrive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestOneDrive(t *testing.T) {
	t.Parallel()

	const (
		driveID = "b!DoXLzayCDEqhLT5H107_ks7hUsDDwPJPlR5iTWhg2CxYuCODHjHUTJOI3xBLJIth"
	)

	clientID := backtest.EnvOrSkip(t, "ONEDRIVE_CLIENT_ID")
	clientSecret := backtest.EnvOrSkip(t, "ONEDRIVE_CLIENT_SECRET")
	tenantID := backtest.EnvOrSkip(t, "ONEDRIVE_TENANT_ID")

	opts := map[string]string{
		"region":             "global",
		"drive_id":           driveID,
		"drive_type":         "personal",
		"tenant":             tenantID,
		"client_credentials": "true",
	}

	blobVFS, fsErr := newVFS("onedrive", clientID, clientSecret, "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, blobVFS)
}

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts := map[string]string{"drive_id": "drive"}
	_, err := parseOpts("account", "password", "", opts)
	require.NoError(t, err)

	assert.NotContains(t, opts, "client_id")
	assert.NotContains(t, opts, "client_secret")
}
