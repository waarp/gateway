package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRequestStatus(t *testing.T) {

	err := os.Setenv("WG_PASSWORD", "pswd")
	if err != nil {
		t.Fatal(err)
	}

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
			Address: server.URL,
		}

		Convey("When calling requestStatus", func() {
			res, err := s.requestStatus()

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
			res, err := s.requestStatus()

			Convey("Then it should return an error", func() {
				So(res, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a server replying anything but 'OK'", t, func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		server := httptest.NewServer(handler)

		s := statusCommand{
			Address: server.URL,
		}

		Convey("When requestStatus is called", func() {
			res, err := s.requestStatus()

			Convey("Then it should return an error", func() {
				So(res, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}
