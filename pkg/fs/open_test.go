package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenFileOverExistingFile(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		flags    Flags
		expected string
	}

	testCases := []testCase{
		{"Truncate", FlagReadWrite | FlagCreate | FlagTruncate, "abc"},
		{"Write only, truncate", FlagWriteOnly | FlagCreate | FlagTruncate, "abc"},
		{"No truncate", FlagWriteOnly | FlagCreate, "abc3456789"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := JoinPath(t.TempDir(), "file.txt")
			require.NoError(t, WriteFullFile(path, []byte("0123456789")))

			file, err := OpenFile(path, tc.flags, FilePerms)
			require.NoError(t, err)

			_, err = file.Write([]byte("abc"))
			require.NoError(t, err)
			require.NoError(t, file.Close())

			content, err := ReadFullFile(path)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(content))
		})
	}
}
