package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
)

type testRESTServer struct {
	*httptest.Server
	db *database.DB
}

func makeTestRESTServer(c convey.C) *testRESTServer {
	db := database.TestDatabase(c)
	logger := logging.NewLogger("rest_test")

	router := mux.NewRouter()
	MakeRESTHandler(logger, db, router)

	server := httptest.NewServer(router)

	return &testRESTServer{
		Server: server,
		db:     db,
	}
}

func (t *testRESTServer) doRequest(method, url string, body io.Reader) *http.Response {
	request, err := http.NewRequest(method, url, body)
	convey.So(err, convey.ShouldBeNil)

	request.SetBasicAuth("admin", "admin_password")

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	convey.So(err, convey.ShouldBeNil)

	convey.Reset(func() { convey.So(response.Body.Close(), convey.ShouldBeNil) })

	return response
}

func (t *testRESTServer) post(url string, body map[string]any) *http.Response {
	rawBody, err := json.Marshal(body)
	convey.So(err, convey.ShouldBeNil)

	return t.doRequest(http.MethodPost, url, bytes.NewReader(rawBody))
}

func (t *testRESTServer) get(url string) *http.Response {
	return t.doRequest(http.MethodGet, url, nil)
}

func (t *testRESTServer) patch(url string, body map[string]any) *http.Response {
	rawBody, err := json.Marshal(body)
	convey.So(err, convey.ShouldBeNil)

	return t.doRequest(http.MethodPatch, url, bytes.NewReader(rawBody))
}

func (t *testRESTServer) put(url string, body map[string]any) *http.Response {
	rawBody, err := json.Marshal(body)
	convey.So(err, convey.ShouldBeNil)

	return t.doRequest(http.MethodPut, url, bytes.NewReader(rawBody))
}

func (t *testRESTServer) delete(url string) *http.Response {
	return t.doRequest(http.MethodDelete, url, nil)
}

func parseBody(body io.Reader) (m map[string]any) {
	decoder := json.NewDecoder(body)
	convey.So(decoder.Decode(&m), convey.ShouldBeNil)

	return m
}

func readBody(body io.Reader) string {
	raw, err := io.ReadAll(body)
	convey.So(err, convey.ShouldBeNil)

	return string(raw)
}

func marshal(tb testing.TB, data any) string {
	tb.Helper()

	raw, err := json.Marshal(data)
	require.NoError(tb, err)

	return string(raw)
}

func replaceURLVar(url, elem string) (string, string) {
	if elem == "" {
		return url, ""
	}

	begin := strings.Index(url, "{")
	end := strings.Index(url, "}")

	urlVar := url[begin+1 : end]
	reqURL := url[:begin] + elem + url[end+1:]

	return reqURL, urlVar
}

func makeRequest(method string, body io.Reader, url, elem string) *http.Request {
	reqURL, urlVar := replaceURLVar(url, elem)
	req := httptest.NewRequest(method, reqURL, body)

	if urlVar != "" {
		req = mux.SetURLVars(req, map[string]string{urlVar: elem})
	}

	return req
}

type testFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (t *testFileInfo) Name() string       { return t.name }
func (t *testFileInfo) Size() int64        { return t.size }
func (t *testFileInfo) Mode() fs.FileMode  { return t.mode }
func (t *testFileInfo) ModTime() time.Time { return t.modTime }
func (t *testFileInfo) IsDir() bool        { return t.mode.IsDir() }
func (t *testFileInfo) Sys() any           { return nil }

func assertBodyMatches(tb testing.TB, body *bytes.Buffer, expected any) {
	tb.Helper()

	rawExpected, err := json.Marshal(expected)
	require.NoError(tb, err)

	assert.JSONEq(tb, string(rawExpected), body.String())
}
