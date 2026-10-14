package tasks

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestZip(tb testing.TB, entries ...string) []byte {
	tb.Helper()

	var buf bytes.Buffer

	writer := zip.NewWriter(&buf)

	for _, name := range entries {
		file, err := writer.Create(name)
		require.NoError(tb, err)

		if strings.HasSuffix(name, "/") {
			continue // directory entry, no content
		}

		_, err = file.Write([]byte("content of " + name))
		require.NoError(tb, err)
	}

	require.NoError(tb, writer.Close())

	return buf.Bytes()
}

func makeTestTar(tb testing.TB, entries ...string) []byte {
	tb.Helper()

	var buf bytes.Buffer

	writer := tar.NewWriter(&buf)

	for _, name := range entries {
		if strings.HasSuffix(name, "/") {
			require.NoError(tb, writer.WriteHeader(&tar.Header{
				Name: name, Mode: 0o700, Typeflag: tar.TypeDir,
			}))

			continue
		}

		content := []byte("content of " + name)
		require.NoError(tb, writer.WriteHeader(&tar.Header{
			Name: name, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg,
		}))

		_, err := writer.Write(content)
		require.NoError(tb, err)
	}

	require.NoError(tb, writer.Close())

	return buf.Bytes()
}

func TestExtractConfinement(t *testing.T) {
	t.Parallel()

	formats := map[string]func(testing.TB, ...string) []byte{
		".zip": makeTestZip,
		".tar": makeTestTar,
	}

	for ext, makeArchive := range formats {
		cases := []string{"../evade.txt", "sub/../../evade.txt"}
		if runtime.GOOS == "windows" {
			cases = append(cases, `..\evade.txt`)
		}

		for _, outside := range cases {
			t.Run(ext+" archive with entry "+outside, func(t *testing.T) {
				t.Parallel()

				root := t.TempDir()
				archive := filepath.Join(root, "archive"+ext)
				require.NoError(t, os.WriteFile(archive, makeArchive(t, "inside.txt", outside), 0o600))

				task := &extractTask{Archive: archive, OutputDir: filepath.Join(root, "out")}
				err := task.extractArchive()

				require.ErrorIs(t, err, ErrExtractInvalidPath)
				assert.NoFileExists(t, filepath.Join(root, "evade.txt"))
			})
		}

		t.Run(ext+" archive with subdirectories", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			archive := filepath.Join(root, "archive"+ext)
			require.NoError(t, os.WriteFile(archive,
				makeArchive(t, "./", "inside.txt", "sub/dir/file.txt", "sub/../back.txt"), 0o600))

			task := &extractTask{Archive: archive, OutputDir: filepath.Join(root, "out")}
			require.NoError(t, task.extractArchive())

			assert.FileExists(t, filepath.Join(root, "out", "inside.txt"))
			assert.FileExists(t, filepath.Join(root, "out", "sub", "dir", "file.txt"))
			assert.FileExists(t, filepath.Join(root, "out", "back.txt"))
		})
	}
}
