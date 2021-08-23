package rest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAddressOverride(t *testing.T) {
	logger := log.NewLogger("rest_addr_ovrd_get_test")

	Convey("Given the address override get handler", t, func() {
		ovrdFile := filepath.Join(os.TempDir(), "rest_test_get_addr_override.ini")
		handler := getAddressOverride(logger)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func() {
			conf.GlobalConfig.LocalOverrides = conf.NewOverride(ovrdFile)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("Given a request with a valid address parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"address": "localhost"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then the body should contain the requested indirection "+
						"in JSON format", func() {
						So(w.Body.String(), ShouldResemble, `{"localhost":"127.0.0.1"}`+"\n")
					})

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain "+
						"'application/json'", func() {
						contentType := w.Header().Get("Content-Type")
						So(contentType, ShouldEqual, "application/json")
					})
				})
			})

			Convey("Given a request with an unknown address parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"address": "unknown"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Not Found'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestListAddressOverride(t *testing.T) {
	logger := log.NewLogger("rest_addr_ovrd_list_test")

	Convey("Given the address override list handler", t, func() {
		ovrdFile := filepath.Join(os.TempDir(), "rest_test_list_addr_override.ini")
		handler := listAddressOverrides(logger)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func() {
			conf.GlobalConfig.LocalOverrides = conf.NewOverride(ovrdFile)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				handler.ServeHTTP(w, r)

				Convey("Then the body should contain the requested indirection "+
					"in JSON format", func() {
					So(w.Body.String(), ShouldBeIn,
						`{"localhost":"127.0.0.1","[::1]":"192.168.1.1"}`+"\n",
						`{"[::1]":"192.168.1.1","localhost":"127.0.0.1"}`+"\n")
				})

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the 'Content-Type' header should contain "+
					"'application/json'", func() {
					contentType := w.Header().Get("Content-Type")
					So(contentType, ShouldEqual, "application/json")
				})
			})
		})
	})
}

func TestAddAddressOverride(t *testing.T) {
	logger := log.NewLogger("rest_addr_ovrd_add_test")

	Convey("Given the address override add handler", t, func() {
		ovrdFile := filepath.Join(os.TempDir(), "rest_test_add_addr_override.ini")
		handler := addAddressOverride(logger)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func() {
			conf.GlobalConfig.LocalOverrides = conf.NewOverride(ovrdFile)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				body := strings.NewReader(`{"localhost":"127.0.0.1",
					"example.com":"8.8.8.8:80"}`)
				r, err := http.NewRequest(http.MethodPost, "", body)
				So(err, ShouldBeNil)
				handler.ServeHTTP(w, r)

				Convey("Then the body should be empty", func() {
					So(w.Body.String(), ShouldBeBlank)
				})

				Convey("Then it should reply 'Created'", func() {
					So(w.Code, ShouldEqual, http.StatusCreated)
				})

				Convey("Then the indirections should have been added to the config", func() {
					So(conf.GlobalConfig.LocalOverrides.ListenAddresses.
						GetRealAddress("localhost"), ShouldEqual, "127.0.0.1")
					So(conf.GlobalConfig.LocalOverrides.ListenAddresses.
						GetRealAddress("example.com"), ShouldEqual, "8.8.8.8:80")
				})
			})
		})
	})
}

func TestDeleteAddressOverride(t *testing.T) {
	logger := log.NewLogger("rest_addr_ovrd_delete_test")

	Convey("Given the address override delete handler", t, func() {
		ovrdFile := filepath.Join(os.TempDir(), "rest_test_delete_addr_override.ini")
		handler := deleteAddressOverride(logger)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func() {
			conf.GlobalConfig.LocalOverrides = conf.NewOverride(ovrdFile)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"localhost", "127.0.0.1"), ShouldBeNil)
			So(conf.GlobalConfig.LocalOverrides.ListenAddresses.AddIndirection(
				"[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("Given a request with a valid address parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"address": "localhost"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then the body should be blank", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should reply 'NoContent'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the indirection should have been deleted", func() {
						So(conf.GlobalConfig.LocalOverrides.ListenAddresses.
							GetRealAddress("localhost"), ShouldBeBlank)
					})
				})
			})

			Convey("Given a request with an unknown address parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"address": "unknown"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Not Found'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
