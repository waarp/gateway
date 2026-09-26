package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityConfigDefaults(t *testing.T) {
	t.Parallel()

	// A configuration file written before the [security] section existed.
	file := filepath.Join(t.TempDir(), "gatewayd.ini")
	require.NoError(t, os.WriteFile(file, []byte("[log]\nLevel = INFO\n"), 0o600))

	config, err := ParseServerConfig(file)
	require.NoError(t, err)

	assert.Equal(t, SecurityConfig{
		MaxAuthFailures:     5,
		AuthFailureWindow:   5 * time.Minute,
		AuthLockDuration:    time.Minute,
		AuthMaxLockDuration: 30 * time.Minute,
	}, config.Security)
}
