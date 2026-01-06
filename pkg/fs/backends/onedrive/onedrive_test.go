package onedrive

import (
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
	"github.com/stretchr/testify/require"
)

// A quick test to check that the most common operations are working properly.
func TestOneDrive(t *testing.T) {
	t.Parallel()

	const (
		driveID = "b!DoXLzayCDEqhLT5H107_ks7hUsDDwPJPlR5iTWhg2CxYuCODHjHUTJOI3xBLJIth"
	)

	clientID := os.Getenv("ONEDRIVE_CLIENT_ID")
	clientSecret := os.Getenv("ONEDRIVE_CLIENT_SECRET")
	tenantID := os.Getenv("ONEDRIVE_TENANT_ID")

	opts := map[string]string{
		"region":             "global",
		"drive_id":           driveID,
		"drive_type":         "personal",
		"tenant":             tenantID,
		"client_credentials": "true",
	}

	blobVFS, fsErr := newVFS("onedrive", clientID, clientSecret, opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, blobVFS)
}
