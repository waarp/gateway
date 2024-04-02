package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/smartystreets/goconvey/convey"

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
