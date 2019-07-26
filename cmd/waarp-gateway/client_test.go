package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	envPassword = "test"
}

func testHandler(expectedMethod, expectedPath string, expectedInput, expectedOutput interface{},
	expectedParam url.Values, expectedCode int) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pswd, ok := r.BasicAuth(); !ok || user != auth.Username || pswd != envPassword {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != expectedPath {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if r.Method != expectedMethod {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if expectedParam != nil {
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for name, param := range expectedParam {
				if !reflect.DeepEqual(param, r.Form[name]) {
					http.Error(w, fmt.Sprintf("Incorrect parameter '%s' expected:'%s' got:'%s'",
						name, param, r.Form[name]), http.StatusBadRequest)
					return
				}
			}
		}

		if expectedInput != nil {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			typ := reflect.TypeOf(expectedInput).Elem()
			input := reflect.New(typ).Interface()

			if err := json.Unmarshal(body, input); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if !reflect.DeepEqual(input, expectedInput) {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(expectedCode)

		if expectedOutput != nil {
			body, err := json.Marshal(expectedOutput)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := w.Write(body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	})
}

func TestRequestStatus(t *testing.T) {
	statuses := admin.Statuses{
		"service1": admin.Status{State: "running"},
		"service2": admin.Status{State: "offline"},
	}

	Convey("Given a server replying correctly", t, func() {
		path := admin.RestURI + admin.StatusURI
		handler := testHandler(http.MethodGet, path, nil, statuses, nil,
			http.StatusOK)
		server := httptest.NewServer(handler)

		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		s := statusCommand{}

		Convey("When calling requestStatus", func() {
			res, err := s.requestStatus(os.Stdin, os.Stdout)

			Convey("Then it should return a valid JSON content and no error", func() {
				So(err, ShouldBeNil)
				So(res, ShouldNotBeNil)
			})
		})
	})

	Convey("Given no server", t, func() {

		auth = ConnectionOptions{}
		s := statusCommand{}

		Convey("When requestStatus is called", func() {
			res, err := s.requestStatus(os.Stdin, os.Stdout)

			Convey("Then it should return an error", func() {
				So(res, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a server replying '401 - Unauthorized'", t, func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		server := httptest.NewServer(handler)

		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		s := statusCommand{}

		Convey("Given that the password is given via an environment variable", func() {

			Convey("When requestStatus is called", func() {
				res, err := s.requestStatus(os.Stdin, os.Stdout)

				Convey("Then it should return an error", func() {
					So(res, ShouldBeNil)
					So(err, ShouldNotBeNil)
				})
			})
		})
	})

	Convey("Given a server replying anything other than '200' or '401'", t, func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		server := httptest.NewServer(handler)

		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		s := statusCommand{}

		Convey("When requestStatus is called", func() {
			res, err := s.requestStatus(os.Stdin, os.Stdout)

			Convey("Then it should return an error", func() {
				So(res, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestShowStatus(t *testing.T) {

	Convey("Given a list of status", t, func() {
		statuses := make(map[string]admin.Status)
		statuses["Test running 1"] = admin.Status{State: "Running", Reason: ""}
		statuses["Test running 2"] = admin.Status{State: "Running", Reason: ""}
		statuses["Test offline 1"] = admin.Status{State: "Offline", Reason: ""}
		statuses["Test offline 2"] = admin.Status{State: "Offline", Reason: ""}
		statuses["Test error 1"] = admin.Status{State: "Error", Reason: "Reason 1"}
		statuses["Test error 2"] = admin.Status{State: "Error", Reason: "Reason 2"}

		Convey("When calling 'showStatus'", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			showStatus(out, statuses)
			_ = out.Close()
			in, err := os.Open(out.Name())
			So(err, ShouldBeNil)
			result, err := ioutil.ReadAll(in)
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the statuses correctly", func() {
				expected := "\n" +
					"Waarp-Gateway services :\n" +
					"[Error]   Test error 1 : Reason 1\n" +
					"[Error]   Test error 2 : Reason 2\n" +
					"[Active]  Test running 1\n" +
					"[Active]  Test running 2\n" +
					"[Offline] Test offline 1\n" +
					"[Offline] Test offline 2\n" +
					"\n"

				So(result, ShouldNotBeNil)
				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
