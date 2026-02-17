package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathUnrooted(t *testing.T) {
	t.Parallel()

	type testCase struct {
		path    string
		expRoot string
		expPath string
	}

	testCases := []testCase{
		{`C:/windows/absolute/path.txt`, `//?/C:/`, `windows/absolute/path.txt`},
		{`C:\windows\absolute\path.txt`, `//?/C:/`, `windows/absolute/path.txt`},
		{`\\host\share\absolute\path.txt`, `//?/UNC/host/share/`, `absolute/path.txt`},
		{`//host/share/absolute/path.txt`, `//?/UNC/host/share/`, `absolute/path.txt`},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			parsed, vfs, err := parseFs(tc.path)
			require.NoError(t, err)

			locFs, ok := vfs.(*LocalFS)
			require.True(t, ok)
			assert.Equal(t, tc.expRoot, locFs.fs().Root())
			assert.Equal(t, tc.expPath, parsed.unrooted())
		})
	}
}
