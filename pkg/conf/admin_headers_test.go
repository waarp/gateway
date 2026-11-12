package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parse "code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
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

	t.Run("A mode of the content security policy left empty", func(t *testing.T) {
		t.Parallel()

		config, err := parseAdminSection(t, "ContentSecurityPolicy =\n")
		require.NoError(t, err)

		assert.Equal(t, CSPReportOnly, config.Admin.ContentSecurityPolicy)
	})

	t.Run("A configuration file written from options that were never read", func(t *testing.T) {
		t.Parallel()

		written := &ServerConfig{}
		parser, err := parse.NewParser(written)
		require.NoError(t, err)

		// The commands that rewrite the file from the configuration they run
		// with replace the options as a whole, the unset ones included.
		*written = ServerConfig{}
		file := filepath.Join(t.TempDir(), "gatewayd.ini")
		require.NoError(t, parser.WriteFile(file))

		read := &ServerConfig{}
		reader, err := parse.NewParser(read)
		require.NoError(t, err)
		require.NoError(t, reader.ParseFile(file))

		assert.Equal(t, CSPReportOnly, read.Admin.ContentSecurityPolicy)
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
