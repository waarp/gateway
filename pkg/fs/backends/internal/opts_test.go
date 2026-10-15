package internal

import (
	"testing"

	_ "github.com/rclone/rclone/backend/azureblob"
	_ "github.com/rclone/rclone/backend/azurefiles"
	_ "github.com/rclone/rclone/backend/googlecloudstorage"
	_ "github.com/rclone/rclone/backend/onedrive"
	_ "github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("Backends", func(t *testing.T) {
		t.Parallel()

		for _, backend := range []string{"s3", "azureblob", "azurefiles", "gcs", "onedrive"} {
			_, err := WithDefaults(backend, configmap.Simple{})
			require.NoError(t, err, backend)
		}
	})

	t.Run("Unknown backend", func(t *testing.T) {
		t.Parallel()

		_, err := WithDefaults("unknown", configmap.Simple{})
		require.Error(t, err)
	})

	t.Run("Values", func(t *testing.T) {
		t.Parallel()

		opts := configmap.Simple{"upload_concurrency": "8"}

		config, err := WithDefaults("s3", opts)
		require.NoError(t, err)

		value, ok := config.Get("force_path_style")
		assert.True(t, ok)
		assert.Equal(t, "true", value, "unset option should take its default value")

		value, ok = config.Get("upload_concurrency")
		assert.True(t, ok)
		assert.Equal(t, "8", value, "set option should keep its value")

		_, ok = config.Get("not_an_option")
		assert.False(t, ok)

		config.Set("token", "new")
		assert.Equal(t, "new", opts["token"], "set should write to the options")
	})
}
