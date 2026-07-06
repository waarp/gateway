package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermsFromString(t *testing.T) {
	t.Parallel()

	for _, tc := range []FileMode{
		0,     // ----------
		0o644, // -rw-r--r--
		0o755, // -rwxr-xr-x
		0o777, // -rwxrwxrwx
	} {
		t.Run(tc.String(), func(t *testing.T) {
			t.Parallel()

			res, err := PermsFromString(tc.String())
			require.NoError(t, err)
			assert.Equal(t, tc, res)
		})
	}
}
