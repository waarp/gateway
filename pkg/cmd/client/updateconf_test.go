package wg

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUpdateconfServer starts a test HTTP server which runs the given check
// function for every incoming request, and points the global "addr" variable
// to it.
func testUpdateconfServer(tb testing.TB, check func(w http.ResponseWriter, r *http.Request)) {
	tb.Helper()

	serv := httptest.NewServer(http.HandlerFunc(check))
	tb.Cleanup(serv.Close)

	host, err := url.Parse(serv.URL)
	require.NoError(tb, err)

	addr = *host
	addr.User = url.UserPassword(testUser, testPswd)
}

func TestUpdateconfImportDefault(t *testing.T) {
	const content = `{"rules":[{"name":"foo"}]}`

	file := writeFile(t, "import.json", content)

	w := newTestOutput()
	command := &UpdateconfImport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		user, pswd, ok := r.BasicAuth()
		assert.True(t, ok, "missing REST credentials")
		assert.Equal(t, testUser, user, "invalid REST username")
		assert.Equal(t, testPswd, pswd, "invalid REST password")

		assert.Equal(t, http.MethodPost, r.Method, "wrong request method")
		assert.Equal(t, updateconfAPIPath, r.URL.Path, "wrong request path")

		assert.Equal(t, "application/json", r.Header.Get("Content-Type"),
			`Then the "Content-Type" header should default to JSON`)
		assert.Equal(t, []string{"all"}, r.Header.Values("Targets"),
			`Then the "Targets" header should default to "all"`)
		assert.Empty(t, r.Header.Get("Dry-Run"), `Then the "Dry-Run" header should NOT be set`)
		assert.Empty(t, r.Header.Get("Reset"), `Then the "Reset" header should NOT be set`)
		assert.Empty(t, r.Header.Get("Restart"), `Then the "Restart" header should NOT be set`)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read the request body")
		assert.Equal(t, content, string(body), "wrong request body")

		rw.WriteHeader(http.StatusCreated)
	})

	require.NoError(t, executeCommand(t, w, command, "--source", file),
		"Then it should not return an error")

	assert.Equal(t, "The configuration was successfully imported.\n", w.String(),
		"Then it should display a success message")
}

func TestUpdateconfImportYAML(t *testing.T) {
	const content = "rules:\n    - name: foo\n"

	file := writeFile(t, "import.yaml", content)

	w := newTestOutput()
	command := &UpdateconfImport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/yaml", r.Header.Get("Content-Type"),
			`Then the "Content-Type" header should be YAML`)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read the request body")
		assert.Equal(t, content, string(body), "wrong request body")

		rw.WriteHeader(http.StatusCreated)
	})

	require.NoError(t, executeCommand(t, w, command, "--source", file),
		"Then it should not return an error")
}

func TestUpdateconfImportArgs(t *testing.T) {
	const content = `{"rules":[]}`

	file := writeFile(t, "import.json", content)

	w := newTestOutput()
	command := &UpdateconfImport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"rules", "clients"}, r.Header.Values("Targets"),
			`Then the "Targets" header should contain the given targets`)
		assert.Equal(t, "true", r.Header.Get("Dry-Run"),
			`Then the "Dry-Run" header should be set`)
		assert.Equal(t, "true", r.Header.Get("Reset"),
			`Then the "Reset" header should be set`)
		assert.Equal(t, "true", r.Header.Get("Restart"),
			`Then the "Restart" header should be set`)

		rw.WriteHeader(http.StatusCreated)
	})

	require.NoError(t, executeCommand(t, w, command,
		"--source", file,
		"--target", "rules",
		"--target", "clients",
		"--dry-run",
		"--reset",
		"--restart",
	), "Then it should not return an error")
}

func TestUpdateconfImportError(t *testing.T) {
	const errMsg = "something went wrong"

	file := writeFile(t, "import.json", `{}`)

	w := newTestOutput()
	command := &UpdateconfImport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, errMsg)
	})

	err := executeCommand(t, w, command, "--source", file)
	require.Error(t, err, "Then it should return an error")
	assert.Contains(t, err.Error(), errMsg, "Then the error should contain the server's message")
}

func TestUpdateconfExportDefault(t *testing.T) {
	const content = `{"rules":[{"name":"foo"}]}`

	w := newTestOutput()
	command := &UpdateconfExport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		user, pswd, ok := r.BasicAuth()
		assert.True(t, ok, "missing REST credentials")
		assert.Equal(t, testUser, user, "invalid REST username")
		assert.Equal(t, testPswd, pswd, "invalid REST password")

		assert.Equal(t, http.MethodGet, r.Method, "wrong request method")
		assert.Equal(t, updateconfAPIPath, r.URL.Path, "wrong request path")

		assert.Equal(t, "application/json", r.Header.Get("Accept"),
			`Then the "Accept" header should default to JSON`)
		assert.Equal(t, []string{"all"}, r.Header.Values("Targets"),
			`Then the "Targets" header should default to "all"`)

		rw.WriteHeader(http.StatusOK)
		fmt.Fprint(rw, content)
	})

	require.NoError(t, executeCommand(t, w, command), "Then it should not return an error")

	assert.Equal(t, content, w.String(),
		"Then it should display the exported configuration")
}

func TestUpdateconfExportArgs(t *testing.T) {
	w := newTestOutput()
	command := &UpdateconfExport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"rules", "clients"}, r.Header.Values("Targets"),
			`Then the "Targets" header should contain the given targets`)

		rw.WriteHeader(http.StatusOK)
	})

	require.NoError(t, executeCommand(t, w, command,
		"--target", "rules",
		"--target", "clients",
	), "Then it should not return an error")
}

func TestUpdateconfExportToFile(t *testing.T) {
	const content = "rules:\n    - name: foo\n"

	dest := filepath.Join(t.TempDir(), "export.yaml")

	w := newTestOutput()
	command := &UpdateconfExport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/yaml", r.Header.Get("Accept"),
			`Then the "Accept" header should be YAML`)

		rw.WriteHeader(http.StatusOK)
		fmt.Fprint(rw, content)
	})

	require.NoError(t, executeCommand(t, w, command, "--file", dest),
		"Then it should not return an error")

	assert.Equal(t, "The configuration was successfully exported.\n", w.String(),
		"Then it should display a success message")

	data, err := os.ReadFile(dest)
	require.NoError(t, err, "failed to read the destination file")
	assert.Equal(t, content, string(data), "wrong destination file content")
}

func TestUpdateconfExportError(t *testing.T) {
	const errMsg = "something went wrong"

	w := newTestOutput()
	command := &UpdateconfExport{}

	testUpdateconfServer(t, func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, errMsg)
	})

	err := executeCommand(t, w, command)
	require.Error(t, err, "Then it should return an error")
	assert.Contains(t, err.Error(), errMsg, "Then the error should contain the server's message")
}
