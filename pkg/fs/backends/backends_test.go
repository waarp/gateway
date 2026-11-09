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

	kinds := []string{"s3", "aws", "azblob", "azureblob", "azfiles", "azurefiles", "gcs", "google", "onedrive", "sharepoint"}

	for _, kind := range kinds {
		lib, ok := fs.LibraryOf(kind)
		require.Truef(t, ok, "%s is described", kind)

		_, err := rclonefs.Find(lib.Name)
		assert.NoErrorf(t, err, "%s: rclone knows %q", kind, lib.Name)
	}

	_, ok := fs.LibraryOf("ftp")
	assert.False(t, ok)
}
