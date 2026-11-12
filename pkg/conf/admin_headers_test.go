package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseAdminSection parses a configuration file whose [admin] section holds
// the given lines.
func parseAdminSection(tb testing.TB, lines string) (*ServerConfig, error) {
	tb.Helper()

	file := filepath.Join(tb.TempDir(), "gatewayd.ini")
	require.NoError(tb, os.WriteFile(file, []byte("[log]\nLevel = INFO\n\n[admin]\n"+lines), 0o600))

	return ParseServerConfig(file)
}

func TestAdminSecurityHeadersOptions(t *testing.T) {
	t.Parallel()

	t.Run("A configuration file written before the options existed", func(t *testing.T) {
		t.Parallel()

		config, err := parseAdminSection(t, "")
		require.NoError(t, err)

		assert.Equal(t, CSPReportOnly, config.Admin.ContentSecurityPolicy)
		assert.Zero(t, config.Admin.HSTSMaxAge)
	})

	t.Run("Each mode of the content security policy", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []CSPMode{CSPReportOnly, CSPEnforce, CSPOff} {
			config, err := parseAdminSection(t, "ContentSecurityPolicy = "+string(mode)+"\n")
			require.NoError(t, err, mode)

			assert.Equal(t, mode, config.Admin.ContentSecurityPolicy)
		}
	})

	t.Run("An unknown mode of the content security policy", func(t *testing.T) {
		t.Parallel()

		_, err := parseAdminSection(t, "ContentSecurityPolicy = strict\n")
		// The options parser reports the error by its text, with the line of the file.
		require.Error(t, err)
		assert.Contains(t, err.Error(), errInvalidCSPMode.Error()+
			` "strict": the allowed values are "report-only", "enforce" and "off"`)
	})

	t.Run("A max age for HTTPS only", func(t *testing.T) {
		t.Parallel()

		config, err := parseAdminSection(t, "HSTSMaxAge = 8760h\n")
		require.NoError(t, err)

		assert.Equal(t, 365*24*time.Hour, config.Admin.HSTSMaxAge)
	})
}
