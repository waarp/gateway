package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseEnvironmentLabel parses a configuration file whose [admin] section
// holds the given lines.
func parseEnvironmentLabel(tb testing.TB, lines string) (*ServerConfig, error) {
	tb.Helper()

	file := filepath.Join(tb.TempDir(), "gatewayd.ini")
	require.NoError(tb, os.WriteFile(file, []byte("[log]\nLevel = INFO\n\n[admin]\n"+lines), 0o600))

	return ParseServerConfig(file)
}

func TestEnvironmentLabel(t *testing.T) {
	t.Parallel()

	longest := strings.Repeat("é", MaxEnvironmentLabelLength)

	t.Run("A configuration file written before the option existed", func(t *testing.T) {
		t.Parallel()

		config, err := parseEnvironmentLabel(t, "")
		require.NoError(t, err)

		assert.Empty(t, config.Admin.EnvironmentLabel)
	})

	t.Run("A label, with the spaces around it left out", func(t *testing.T) {
		t.Parallel()

		for line, expected := range map[string]EnvironmentLabel{
			"EnvironmentLabel = Production":      "Production",
			"EnvironmentLabel =   Pré-prod 2   ": "Pré-prod 2",
			"EnvironmentLabel = " + longest:      EnvironmentLabel(longest),
			"EnvironmentLabel =":                 "",
		} {
			config, err := parseEnvironmentLabel(t, line+"\n")
			require.NoError(t, err, line)

			assert.Equal(t, expected, config.Admin.EnvironmentLabel, line)
		}
	})

	t.Run("A label that is too long", func(t *testing.T) {
		t.Parallel()

		_, err := parseEnvironmentLabel(t, "EnvironmentLabel = "+longest+"é\n")
		require.Error(t, err)
		// The options parser reports the error by its text, with the line of the file.
		assert.Contains(t, err.Error(), errInvalidEnvironmentLabel.Error()+": 33 characters, 32 at most")
	})

	t.Run("A label with a character that cannot be printed", func(t *testing.T) {
		t.Parallel()

		var label EnvironmentLabel

		for _, value := range []string{"Pro\tduction", "Pro\nduction", "Pro​duction", "Pro duction"} {
			err := label.UnmarshalFlag(value)
			require.ErrorIs(t, err, errInvalidEnvironmentLabel, "%q", value)
			assert.Contains(t, err.Error(), "a character that cannot be printed")
		}

		assert.Empty(t, label, "a refused label is not kept")
	})
}
