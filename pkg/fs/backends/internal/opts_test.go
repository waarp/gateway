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
)

func TestMakeOptions(t *testing.T) {
	t.Parallel()

	t.Run("Backends", func(t *testing.T) {
		t.Parallel()

		for _, backend := range []string{"s3", "azureblob", "azurefiles", "gcs", "onedrive"} {
			assert.NotPanics(t, func() {
				MakeOptions(backend, map[string]string{}, map[string]string{})
			})
		}
	})

	t.Run("Unknown backend", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			MakeOptions("unknown", configmap.Simple{}, configmap.Simple{})
		})
	})

	t.Run("Values", func(t *testing.T) {
		t.Parallel()

		opts := configmap.Simple{"upload_concurrency": "8"}
		overrides := configmap.Simple{}

		config := MakeOptions("s3", opts, overrides)

		value, ok := config.Get("force_path_style")
		assert.True(t, ok)
		assert.Equal(t, "true", value, "unset option should take its default value")

		value, ok = config.Get("upload_concurrency")
		assert.True(t, ok)
		assert.Equal(t, "8", value, "set option should keep its value")

		_, ok = config.Get("not_an_option")
		assert.False(t, ok)

		config.Set("token", "new")
		assert.Equal(t, "new", overrides["token"], "set should write to the overrides")
	})
}
