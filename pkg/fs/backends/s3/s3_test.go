package s3

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal/backtest"
)

// A quick test to check that the most common operations are working properly.
func TestS3(t *testing.T) {
	const bucket = "waarp-gateway-tests"

	backtest.EnvOrSkip(t, "AWS_ACCESS_KEY_ID")
	backtest.EnvOrSkip(t, "AWS_SECRET_ACCESS_KEY")

	opts := map[string]string{
		"env_auth": "true",
		"bucket":   bucket,
		"provider": "AWS",
		"region":   "eu-west-3",
	}

	s3VFS, fsErr := newVFS("s3", "", "", opts)
	require.NoError(t, fsErr)

	backtest.TestVFS(t, s3VFS)
}

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts := map[string]string{"bucket": "name"}
	_, _, err := parseOpts("account", "password", opts)
	require.NoError(t, err)

	assert.NotContains(t, opts, "access_key_id")
	assert.NotContains(t, opts, "secret_access_key")
}

// Without an explicit "force_path_style", a provider whose quirk enables path
// style access must address the bucket in the URL path (the rclone default),
// not as a sub-domain of the endpoint.
func TestPathStyleDefault(t *testing.T) {
	t.Parallel()

	paths := make(chan string, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	opts := map[string]string{
		"provider": "Minio",
		// a host name, since an IP address always gets a path style URL
		"endpoint": strings.Replace(server.URL, "127.0.0.1", "localhost", 1),
		"bucket":   "bucket",
	}

	s3VFS, err := newVFS(t.Name(), "", "", opts)
	require.NoError(t, err)

	t.Cleanup(s3VFS.Shutdown)

	_, statErr := s3VFS.Stat("file.txt")
	require.Error(t, statErr)

	select {
	case path := <-paths:
		assert.True(t, strings.HasPrefix(path, "/bucket"),
			"the bucket should be in the URL path, got %q", path)
	default:
		assert.Fail(t, "the endpoint was never called (virtual-hosted style address)")
	}
}
