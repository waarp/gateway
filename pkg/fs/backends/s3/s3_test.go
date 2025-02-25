package s3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// A quick test to check that the most common operations are working properly.
func TestS3(t *testing.T) {
	const bucket = "waarp-gateway-tests"

	opts := map[string]string{
		"bucket": bucket,
	}

	s3fs, fsErr := newS3FSWithRoot("s3", "", "", "", opts)
	require.NoError(t, fsErr)

	const (
		dirName    = "foo"
		filename   = "bar"
		filepath   = dirName + "/" + filename
		expContent = "hello world"
	)

	require.NoError(t, s3fs.Mkdir(dirName, 0o700))

	file, opErr := s3fs.OpenFile(filepath, fs.FlagReadWrite|fs.FlagCreate|fs.FlagTruncate, 0o600)
	require.NoError(t, opErr)

	t.Cleanup(func() {
		require.NoError(t, file.Close())

		cont, rErr := s3fs.ReadFile(filepath)
		require.NoError(t, rErr)
		require.Equal(t, expContent, string(cont))

		require.NoError(t, s3fs.Remove(filepath))
		require.NoError(t, s3fs.Remove(dirName))
	})

	_, rErr := file.Write([]byte(expContent))
	require.NoError(t, rErr)
}
