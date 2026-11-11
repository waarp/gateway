package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The [analytics] section: the periods of the disk measure and of the survey
// of the directories, and the cap of the survey.
func TestAnalyticsConfig(t *testing.T) {
	t.Run("when creating a new configuration file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "test.ini")
		require.NoError(t, CreateServerConfig(filePath))

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "[analytics]")

		for _, opt := range []string{"DiskMeasurePeriod", "DiskSurveyPeriod", "DiskSurveyMaxEntries"} {
			assert.Contains(t, string(content), opt+" =")
		}
	})

	t.Run("when the options are set", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "test.ini")
		require.NoError(t, os.WriteFile(filePath, []byte(`[analytics]
DiskMeasurePeriod = 1m
DiskSurveyPeriod = 10m
DiskSurveyMaxEntries = 500
`), 0o600))

		config, err := ParseServerConfig(filePath)
		require.NoError(t, err)

		assert.Equal(t, time.Minute, config.Analytics.DiskMeasurePeriod)
		assert.Equal(t, 10*time.Minute, config.Analytics.DiskSurveyPeriod)
		assert.Equal(t, 500, config.Analytics.DiskSurveyMaxEntries)
	})

	t.Run("when the options are left out", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "test.ini")
		require.NoError(t, os.WriteFile(filePath, []byte("[log]\nLevel = INFO\n"), 0o600))

		config, err := ParseServerConfig(filePath)
		require.NoError(t, err)

		assert.Equal(t, 30*time.Second, config.Analytics.DiskMeasurePeriod)
		assert.Equal(t, 5*time.Minute, config.Analytics.DiskSurveyPeriod)
		assert.Equal(t, 10000, config.Analytics.DiskSurveyMaxEntries)
	})
}
