package rest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestRouterChange(t *testing.T) {
	t.Parallel()

	Convey("Given a gateway router", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "test_router")

		router := mux.NewRouter()
		MakeRESTHandler(logger, db, router)

		serv := httptest.NewServer(router)

		Convey("Given a request made to the server", func() {
			req, err := http.NewRequest(http.MethodGet, serv.URL+"/about", nil)
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
				route := router.Get("GET /about")
				newStatus := http.StatusResetContent

				So(route, ShouldNotBeNil)

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

func TestPathEscape(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := logtest.GetTestLogger(t)

	router := mux.NewRouter()
	MakeRESTHandler(logger, db, router)
	server := httptest.NewServer(router)
	defer server.Close()

	existing := &model.Rule{Name: "existing/rule", IsSend: false}
	require.NoError(t, db.Insert(existing).Run())

	URL := server.URL + "/rules/" + url.PathEscape(existing.Name) + "/receive"
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin_password")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
