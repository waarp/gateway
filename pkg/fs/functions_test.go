package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestGlob(t *testing.T) {
	root := t.TempDir()
	fs, fsErr := NewLocalFS(root)
	require.NoError(t, fsErr)

	filepath1 := filepath.ToSlash(filepath.Join(root, "bar.txt"))
	filepath2 := filepath.ToSlash(filepath.Join(root, "foo.txt"))

	require.NoError(t, os.WriteFile(filepath1, []byte("bar"), 0o700))
	require.NoError(t, os.WriteFile(filepath2, []byte("foo"), 0o700))

	pattern := &types.FSPath{
		Backend: "",
		Path:    filepath.ToSlash(filepath.Join(root, "*.txt")),
	}

	matches, globErr := Glob(fs, pattern)
	require.NoError(t, globErr)

	require.Len(t, matches, 2)
	assert.Empty(t, matches[0].Backend)
	assert.Empty(t, matches[1].Backend)
	assert.Equal(t, filepath1, matches[0].Path)
	assert.Equal(t, filepath2, matches[1].Path)
}
