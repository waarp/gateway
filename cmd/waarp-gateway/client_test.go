package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"code.waarp.fr/waarp/gateway-ng/pkg/admin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRequestStatus(t *testing.T) {

	Convey("Given a server replying correctly", t, func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("{}"))
			if err != nil {
				panic(err.Error())
			}
		})
		server := httptest.NewServer(handler)

		s := statusCommand{
			Address:     server.URL,
			Username:    "test",
			envPassword: "test",
		}

		Convey("When calling requestStatus", func() {
			res, err := s.requestStatus(os.Stdin, os.Stdout)

			Convey("Then it should return a JSON http.Response and no error", func() {
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, http.StatusOK)
				So(res.Header.Get("Content-Type"), ShouldEqual, "application/json")
				body, err := ioutil.ReadAll(res.Body)
				So(err, ShouldBeNil)
				So(json.Valid(body), ShouldBeTrue)
			})
		})
	})

	Convey("Given no server", t, func() {

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

		s := statusCommand{
			Address:     server.URL,
			Username:    "test",
			envPassword: "test",
		}

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

		s := statusCommand{
			Address:     server.URL,
			Username:    "test",
			envPassword: "test",
		}

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
			w := bytes.Buffer{}
			showStatus(&w, statuses)
			result := w.String()

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
				So(result, ShouldEqual, expected)
			})
		})
	})
}
