package wg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUser = "foo"
	testPswd = "bar"
)

type expectedRequest struct {
	method, path string
	values       url.Values
	body         map[string]any
}

type expectedResponse struct {
	status  int
	headers http.Header
	body    map[string]any
}

func testServer(tb testing.TB, expected *expectedRequest, result *expectedResponse) {
	tb.Helper()

	if expected.values == nil {
		expected.values = url.Values{}
	}

	check := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pswd, ok := r.BasicAuth()
		assert.True(tb, ok, "missing REST credentials")
		assert.Equal(tb, testUser, user, "invalid REST username")
		assert.Equal(tb, testPswd, pswd, "invalid REST password")

		assert.Equal(tb, expected.method, r.Method, "wrong request method")
		assert.Equal(tb, expected.path, r.URL.Path, "wrong request path")
		assert.Equal(tb, expected.values, r.URL.Query(), "wrong URL parameters")

		if expected.body != nil {
			var body map[string]any

			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&body)
			require.NoError(tb, err, "failed to parse JSON body")

			assert.Equal(tb, expected.body, body, "wrong request body")
		} else {
			content, err := io.ReadAll(r.Body)
			require.NoError(tb, err, "failed to read request body")

			assert.Empty(tb, content, "expected no body")
		}

		for head, vals := range result.headers {
			for i := range vals {
				w.Header().Add(head, vals[i])
			}
		}

		if result.body != nil {
			respBody, err := json.MarshalIndent(result.body, "", "  ")
			require.NoError(tb, err, "failed to marshal JSON response body")

			defer fmt.Fprint(w, string(respBody))
		}

		w.WriteHeader(result.status)
	})

	serv := httptest.NewServer(check)
	tb.Cleanup(serv.Close)

	if path := result.headers.Get("Location"); path != "" {
		result.headers.Set("Location", serv.URL+path)
	}

	host, err := url.Parse(serv.URL)
	require.NoError(tb, err)

	addr = host
	addr.User = url.UserPassword(testUser, testPswd)
}
