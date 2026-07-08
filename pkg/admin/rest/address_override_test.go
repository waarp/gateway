package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf/conftest"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestListAddressOverride(t *testing.T) {
	t.Parallel()

	Convey("Given the address override list handler", t, func(c C) {
		db := &database.DB{}
		logger := testhelpers.TestLogger(c, "rest_addr_ovrd_list_test")
		handler := listAddressOverrides(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func(c C) {
			conftest.InitTestOverrides(c, db)
			So(db.Config.Overrides.AddIndirection("localhost", "127.0.0.1"), ShouldBeNil)
			So(db.Config.Overrides.AddIndirection("[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil) //nolint:noctx //this is a test
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
	t.Parallel()

	Convey("Given the address override add handler", t, func(c C) {
		db := &database.DB{}
		logger := testhelpers.TestLogger(c, "rest_addr_ovrd_add_test")
		handler := addAddressOverride(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func(c C) {
			conftest.InitTestOverrides(c, db)
			So(db.Config.Overrides.AddIndirection("[::1]", "192.168.1.1"), ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				body := strings.NewReader(`{"localhost":"127.0.0.1",
					"example.com:443":"8.8.8.8:80"}`)
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
					So(db.Config.Overrides.GetIndirection("localhost"), ShouldEqual, "127.0.0.1")
					So(db.Config.Overrides.GetIndirection("example.com:443"), ShouldEqual, "8.8.8.8:80")
				})
			})
		})
	})
}

func TestDeleteAddressOverride(t *testing.T) {
	t.Parallel()

	Convey("Given the address override delete handler", t, func(c C) {
		db := &database.DB{}
		logger := testhelpers.TestLogger(c, "rest_addr_ovrd_delete_test")
		handler := deleteAddressOverride(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a configuration with some address indirections", func(c C) {
			conftest.InitTestOverrides(c, db)
			So(db.Config.Overrides.AddIndirection("localhost", "127.0.0.1"), ShouldBeNil)
			So(db.Config.Overrides.AddIndirection("[::1]", "192.168.1.1"), ShouldBeNil)

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
						So(db.Config.Overrides.GetIndirection("localhost"), ShouldBeBlank)
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
