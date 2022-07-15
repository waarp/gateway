package wg

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"

	"github.com/smartystreets/goconvey/convey"
)

type expectedRequest struct {
	method, path string
	values       url.Values
	body         string
}

type expectedResponse struct {
	status  int
	headers http.Header
	body    string
}

func testServer(expected *expectedRequest, result *expectedResponse) {
	if expected.values == nil {
		expected.values = url.Values{}
	}

	check := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expected.method {
			http.Error(w, fmt.Sprintf("wrong method, expected %s, got %s",
				expected.method, r.Method), http.StatusMethodNotAllowed)

			return
		}

		if r.URL.Path != expected.path {
			http.Error(w, fmt.Sprintf("resource not found, expected %s, got %s",
				expected.path, r.URL.Path), http.StatusNotFound)

			return
		}

		if !reflect.DeepEqual(r.URL.Query(), expected.values) {
			http.Error(w, fmt.Sprintf("bad arguments, expected %s, got %s",
				expected.values, r.URL.Query()), http.StatusBadRequest)

			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read the request body: %v", err),
				http.StatusInternalServerError)

			return
		}

		if body := string(b); body != expected.body {
			http.Error(w, fmt.Sprintf("bad body, expected %s, got %s",
				expected.body, body), http.StatusBadRequest)

			return
		}

		for head, vals := range result.headers {
			for i := range vals {
				w.Header().Add(head, vals[i])
			}
		}

		w.WriteHeader(result.status)
		fmt.Fprint(w, result.body)
	})

	serv := httptest.NewServer(check)

	convey.Reset(func() {
		serv.CloseClientConnections()
		serv.Close()
	})

	host, err := url.Parse(serv.URL)
	convey.So(err, convey.ShouldBeNil)

	addr = host
}
