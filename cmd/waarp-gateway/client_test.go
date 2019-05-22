package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMakeRequest(t *testing.T) {

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

		Convey("When calling makeRequest", func() {
			res, err := s.makeRequest()

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

		s := statusCommand{
			Address:  "",
			Username: "",
			Password: "",
		}

		Convey("When makeRequest is called", func() {
			res, err := s.makeRequest()

			Convey("Then it should return an error", func() {
				So(res, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a server replying 'Unauthorized'", t, func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		server := httptest.NewServer(handler)

		s := statusCommand{
			Address:  server.URL,
			Username: "admin",
			Password: "incorrect_password",
		}

		Convey("When makeRequest is called", func() {
			res, err := s.makeRequest()

			Convey("Then it should return an http.response with code 401", func() {
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})
		})
	})

	Convey("Given a server replying anything but 'OK'", t, func() {
		status := http.StatusInternalServerError
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})
		server := httptest.NewServer(handler)

		s := statusCommand{
			Address:  server.URL,
			Username: "admin",
			Password: "incorrect_password",
		}

		Convey("When makeRequest is called", func() {
			res, err := s.makeRequest()

			Convey("Then it should return an http.response with the corresponding code", func() {
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, status)
			})
		})
	})
}
