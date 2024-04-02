package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestRouterChange(t *testing.T) {
	Convey("Given a gateway router", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "test_router")
		router := mux.NewRouter()

		MakeRESTHandler(logger, db, router)

		serv := httptest.NewServer(router)

		Convey("Given a request made to the server", func() {
			req, err := http.NewRequest(http.MethodGet, serv.URL+"/api/users", nil)
			So(err, ShouldBeNil)

			req.SetBasicAuth("admin", "admin_password")

			Convey("When sending the request", func() {
				//nolint:bodyclose //false positive, the body IS closed
				resp, err := http.DefaultClient.Do(req)
				So(err, ShouldBeNil)
				defer resp.Body.Close()

				Convey("Then it should have called the default handler", func() {
					So(resp.StatusCode, ShouldEqual, http.StatusOK)
				})
			})

			Convey("When changing the target handler", func() {
				route := router.Get("GET /api/users")
				newStatus := http.StatusResetContent

				route.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(newStatus)
				}))

				Convey("When sending the request", func() {
					//nolint:bodyclose //false positive, the body IS closed
					resp, err := http.DefaultClient.Do(req)
					So(err, ShouldBeNil)
					defer resp.Body.Close()

					Convey("Then it should have called the new handler", func() {
						So(resp.StatusCode, ShouldEqual, newStatus)
					})
				})
			})
		})
	})
}
