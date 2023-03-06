package wg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"

	"github.com/smartystreets/goconvey/convey"
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

func testServer(expected *expectedRequest, result *expectedResponse) {
	if expected.values == nil {
		expected.values = url.Values{}
	}

	check := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pswd, ok := r.BasicAuth(); !ok {
			http.Error(w, "missing REST credentials", http.StatusUnauthorized)

			return
		} else if user != testUser || pswd != testPswd {
			http.Error(w, fmt.Sprintf("invalid REST credentials (expected %q and %q, got %q and %q)",
				testUser, testPswd, user, pswd), http.StatusUnauthorized)

			return
		}

		if r.Method != expected.method {
			http.Error(w, fmt.Sprintf("wrong method, expected %q, got %q",
				expected.method, r.Method), http.StatusMethodNotAllowed)

			return
		}

		if r.URL.Path != expected.path {
			http.Error(w, fmt.Sprintf("resource not found, expected %q, got %q",
				expected.path, r.URL.Path), http.StatusNotFound)

			return
		}

		if !reflect.DeepEqual(r.URL.Query(), expected.values) {
			http.Error(w, fmt.Sprintf("bad arguments, expected %v, got %v",
				expected.values, r.URL.Query()), http.StatusBadRequest)

			return
		}

		if expected.body != nil {
			var body map[string]any

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&body); err != nil {
				http.Error(w, fmt.Sprintf("expected a JSON input in body: %v", err),
					http.StatusBadRequest)

				return
			}

			if !reflect.DeepEqual(body, expected.body) {
				http.Error(w, fmt.Sprintf("bad JSON body, expected %v, got %v",
					expected.body, body), http.StatusBadRequest)

				return
			}
		} else {
			if cont, err := io.ReadAll(r.Body); err != nil {
				http.Error(w, fmt.Sprintf("error while reading the request body: %v",
					err), http.StatusInternalServerError)

				return
			} else if len(cont) != 0 {
				http.Error(w, fmt.Sprintf("expected no body, got %q", string(cont)),
					http.StatusBadRequest)

				return
			}
		}

		for head, vals := range result.headers {
			for i := range vals {
				w.Header().Add(head, vals[i])
			}
		}

		if result.body != nil {
			respBody, err := json.MarshalIndent(result.body, "", "  ")
			if err != nil {
				http.Error(w, fmt.Sprintf("error while writing the reesponse body: %v",
					err), http.StatusInternalServerError)

				return
			}

			defer fmt.Fprint(w, string(respBody))
		}

		w.WriteHeader(result.status)
	})

	serv := httptest.NewServer(check)

	convey.Reset(func() {
		serv.CloseClientConnections()
		serv.Close()
	})

	host, err := url.Parse(serv.URL)
	convey.So(err, convey.ShouldBeNil)

	addr = host
	addr.User = url.UserPassword(testUser, testPswd)
}
