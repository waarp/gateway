package backends

import (
	"testing"

	rclonefs "github.com/rclone/rclone/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// Every cloud type, alias included, is described with the storage of rclone
// it is built on, which rclone knows.
func TestLibraries(t *testing.T) {
	t.Parallel()

	aliases := []string{"s3", "aws", "azblob", "azureblob", "azfiles", "azurefiles", "gcs", "google", "onedrive", "sharepoint"}

	for _, alias := range aliases {
		back, ok := fs.GetBackend(alias)
		require.Truef(t, ok, "%s is described", alias)

		_, err := rclonefs.Find(back.Type)
		assert.NoErrorf(t, err, "%s: rclone knows %q", alias, back.Type)
	}

	_, ok := fs.GetBackend("ftp")
	assert.False(t, ok)
}
