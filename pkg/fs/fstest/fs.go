package fstest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoot(tb testing.TB) *os.Root {
	tb.Helper()
	root, err := os.OpenRoot(tb.TempDir())
	require.NoError(tb, err)
	tb.Cleanup(func() { require.NoError(tb, root.Close()) })

	return root
}

func OpenFile(tb testing.TB, root *os.Root, name string) (*os.File, error) {
	tb.Helper()
	file, err := root.Open(name)
	require.NoError(tb, err)
	tb.Cleanup(func() { require.NoError(tb, file.Close()) })

	return file, nil
}

func CreateFile(tb testing.TB, root *os.Root, name string) (*os.File, error) {
	tb.Helper()
	file, err := root.Create(name)
	require.NoError(tb, err)
	tb.Cleanup(func() { require.NoError(tb, file.Close()) })

	return file, nil
}
