package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAddRemoveAuth(t *testing.T) {
	Convey("Given a database with agents & accounts in it", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_add-del_auth")
		db := database.TestDatabase(c)

		server := &model.LocalAgent{
			Name: "server", Protocol: testProto1,
			Address: types.Addr("1.1.1.1", 1),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		partner := &model.RemoteAgent{
			Name: "partner", Protocol: testProto2,
			Address: types.Addr("2.2.2.2", 2),
		}
		So(db.Insert(partner).Run(), ShouldBeNil)

		locAcc := &model.LocalAccount{LocalAgentID: server.ID, Login: "locAcc"}
		So(db.Insert(locAcc).Run(), ShouldBeNil)

		remAcc := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "remAcc"}
		So(db.Insert(remAcc).Run(), ShouldBeNil)

		Convey("When adding an auth method to the server", func() {
			const (
				credName = "pswd"
				credType = auth.Password
				credVal  = "sesame"
			)

			body := map[string]any{
				"name":  credName,
				"type":  credType,
				"value": credVal,
			}

			req1 := httptest.NewRequest(http.MethodPost, ServerAuthsPath,
				utils.ToJSONBody(body))
			req1 = mux.SetURLVars(req1, map[string]string{"server": server.Name})

			resp1 := httptest.NewRecorder()

			addServerCred(logger, db)(resp1, req1)

			Convey("Then it should have added the authentication method", func() {
				So(resp1.Code, ShouldEqual, http.StatusCreated)

				var agAuth model.Credential
				So(db.Get(&agAuth, "local_agent_id=?", server.ID).Run(), ShouldBeNil)
				So(agAuth, ShouldResemble, model.Credential{
					ID:           1,
					LocalAgentID: utils.NewNullInt64(server.ID),
					Name:         credName,
					Type:         credType,
					Value:        credVal,
				})

				Convey("When removing the server's auth method", func() {
					req2 := httptest.NewRequest(http.MethodDelete, ServerAuthPath, nil)
					req2 = mux.SetURLVars(req2, map[string]string{
						"server":     server.Name,
						"credential": agAuth.Name,
					})

					resp2 := httptest.NewRecorder()

					removeServerCred(logger, db)(resp2, req2)

					Convey("Then it should have removed the authentication method", func() {
						So(resp2.Code, ShouldEqual, http.StatusNoContent)

						var auths model.Credentials
						So(db.Select(&auths).Where("local_agent_id=?",
							server.ID).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})
				})
			})
		})

		Convey("When adding an auth method to the partner", func() {
			const (
				credName = "pswd"
				credType = auth.PasswordHash
				credVal  = "sesame"
			)

			body := map[string]any{
				"name":  credName,
				"type":  credType,
				"value": credVal,
			}

			req1 := httptest.NewRequest(http.MethodPost, PartnerCredsPath,
				utils.ToJSONBody(body))
			req1 = mux.SetURLVars(req1, map[string]string{"partner": partner.Name})

			resp1 := httptest.NewRecorder()

			addPartnerCred(logger, db)(resp1, req1)

			Convey("Then it should have added the authentication method", func() {
				So(resp1.Code, ShouldEqual, http.StatusCreated)

				var agAuth model.Credential
				So(db.Get(&agAuth, "remote_agent_id=?", partner.ID).Run(), ShouldBeNil)
				So(agAuth.Name, ShouldEqual, credName)
				So(agAuth.Type, ShouldEqual, credType)

				authRes, authErr := partner.Authenticate(db, credType, credVal)
				So(authErr, ShouldBeNil)
				So(authRes.Success, ShouldBeTrue)

				Convey("When removing the partner's auth method", func() {
					req2 := httptest.NewRequest(http.MethodDelete, PartnerCredPath, nil)
					req2 = mux.SetURLVars(req2, map[string]string{
						"partner":    partner.Name,
						"credential": agAuth.Name,
					})

					resp2 := httptest.NewRecorder()

					removePartnerCred(logger, db)(resp2, req2)

					Convey("Then it should have removed the authentication method", func() {
						So(resp2.Code, ShouldEqual, http.StatusNoContent)

						var auths model.Credentials
						So(db.Select(&auths).Where("remote_agent_id=?",
							partner.ID).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})
				})
			})
		})

		Convey("When adding an auth method to the local account", func() {
			const (
				credName = "pswd"
				credType = auth.PasswordHash
				credVal  = "sesame"
			)

			body := map[string]any{
				"name":  credName,
				"type":  credType,
				"value": credVal,
			}

			req1 := httptest.NewRequest(http.MethodPost, LocAccCredsPath,
				utils.ToJSONBody(body))
			req1 = mux.SetURLVars(req1, map[string]string{
				"server":        server.Name,
				"local_account": locAcc.Login,
			})

			resp1 := httptest.NewRecorder()

			addLocAccAuth(logger, db)(resp1, req1)

			Convey("Then it should have added the authentication method", func() {
				So(resp1.Code, ShouldEqual, http.StatusCreated)

				var accAuth model.Credential
				So(db.Get(&accAuth, "local_account_id=?", locAcc.ID).Run(), ShouldBeNil)
				So(accAuth.Name, ShouldEqual, credName)
				So(accAuth.Type, ShouldEqual, credType)

				authRes, authErr := locAcc.Authenticate(db, credType, credVal)
				So(authErr, ShouldBeNil)
				So(authRes.Success, ShouldBeTrue)

				Convey("When removing the account's auth method", func() {
					req2 := httptest.NewRequest(http.MethodDelete, LocAccCredPath, nil)
					req2 = mux.SetURLVars(req2, map[string]string{
						"server":        server.Name,
						"local_account": locAcc.Login,
						"credential":    accAuth.Name,
					})

					resp2 := httptest.NewRecorder()

					removeLocAccAuth(logger, db)(resp2, req2)

					Convey("Then it should have removed the authentication method", func() {
						So(resp2.Code, ShouldEqual, http.StatusNoContent)

						var auths model.Credentials
						So(db.Select(&auths).Where("local_account_id=?",
							locAcc.ID).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})
				})
			})
		})

		Convey("When adding an auth method to the remote account", func() {
			const (
				credName = "pswd"
				credType = auth.Password
				credVal  = "sesame"
			)

			body := map[string]any{
				"name":  credName,
				"type":  credType,
				"value": credVal,
			}

			req1 := httptest.NewRequest(http.MethodPost, RemAccCredsPath,
				utils.ToJSONBody(body))
			req1 = mux.SetURLVars(req1, map[string]string{
				"partner":        partner.Name,
				"remote_account": remAcc.Login,
			})

			resp1 := httptest.NewRecorder()

			addRemAccCred(logger, db)(resp1, req1)

			Convey("Then it should have added the authentication method", func() {
				So(resp1.Code, ShouldEqual, http.StatusCreated)

				var accAuth model.Credential
				So(db.Get(&accAuth, "remote_account_id=?", remAcc.ID).Run(), ShouldBeNil)
				So(accAuth, ShouldResemble, model.Credential{
					ID:              1,
					Name:            credName,
					RemoteAccountID: utils.NewNullInt64(remAcc.ID),
					Type:            credType,
					Value:           credVal,
				})

				Convey("When removing the account's auth method", func() {
					req2 := httptest.NewRequest(http.MethodDelete, RemAccCredPath, nil)
					req2 = mux.SetURLVars(req2, map[string]string{
						"partner":        partner.Name,
						"remote_account": remAcc.Login,
						"credential":     accAuth.Name,
					})

					resp2 := httptest.NewRecorder()

					removeRemAccCred(logger, db)(resp2, req2)

					Convey("Then it should have removed the authentication method", func() {
						So(resp2.Code, ShouldEqual, http.StatusNoContent)

						var auths model.Credentials
						So(db.Select(&auths).Where("remote_account_id=?",
							remAcc.ID).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})
				})
			})
		})
	})
}
