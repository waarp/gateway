package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestAuth(t *testing.T) {
	const (
		login    = "foobar"
		password = "sesame"
	)

	Convey("Given a test HTTP server", t, func(c C) {
		db := database.TestDatabase(c)

		locAgent := &model.LocalAgent{
			Name:     "http_test_auth",
			Address:  types.Addr("localhost", 0),
			Protocol: HTTP,
		}
		So(db.Insert(locAgent).Run(), ShouldBeNil)

		server := &httpService{db: db, agent: locAgent}
		So(server.Start(), ShouldBeNil)

		Reset(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			So(server.Stop(ctx), ShouldBeNil)
		})

		locAccount := &model.LocalAccount{
			LocalAgentID: locAgent.ID,
			Login:        login,
		}

		So(db.Insert(locAccount).Run(), ShouldBeNil)
		So(db.Insert(&model.Credential{
			LocalAccountID: utils.NewNullInt64(locAccount.ID),
			Type:           auth.Password,
			Value:          password,
		}).Run(), ShouldBeNil)

		req, reqErr := http.NewRequest(http.MethodGet, "http://"+server.serv.Addr, nil)
		So(reqErr, ShouldBeNil)

		req.SetBasicAuth(login, password)

		w := httptest.NewRecorder()

		Convey("Given a normal account", func() {
			Convey("When logging in", func() {
				req.RemoteAddr = "127.0.0.1:80"

				Convey("Then it should succeed", func() {
					_, ok := server.checkAuthent(w, req)
					So(ok, ShouldBeTrue)
				})
			})
		})

		Convey("Given an IP-restricted account", func() {
			locAccount.IPAddresses = []string{"1.2.3.4"}
			So(db.Update(locAccount).Run(), ShouldBeNil)

			Convey("When logging in from the correct IP", func() {
				req.RemoteAddr = "1.2.3.4:80"

				Convey("Then it should succeed", func() {
					_, ok := server.checkAuthent(w, req)
					So(ok, ShouldBeTrue)
				})
			})

			Convey("When logging in from an unauthorized IP", func() {
				req.RemoteAddr = "9.8.7.6:80"

				Convey("Then it should fail", func() {
					_, ok := server.checkAuthent(w, req)
					So(ok, ShouldBeFalse)
					So(w.Code, ShouldEqual, http.StatusUnauthorized)
					So(w.Body.String(), ShouldEqual, "Unauthorized IP address\n")
				})
			})
		})
	})
}
