package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAuthorizeRule(t *testing.T) {
	logger := log.NewLogger("rest_auth_rule_logger")

	Convey("Given a database with 1 rule", t, func() {
		db := database.GetTestDatabase()
		rule := &model.Rule{
			Name: "rule",
			Path: "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		w := httptest.NewRecorder()

		test := func(handler http.Handler, targetType, targetName string,
			expectedResult model.RuleAccess) {

			Convey("When sending the request to the handler", func() {
				r, err := http.NewRequest(http.MethodPut, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{targetType: targetName,
					"rule": rule.Name, "direction": ruleDirection(rule)})
				handler.ServeHTTP(w, r)

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the response body should state that access "+
					"to the rule is now restricted", func() {
					So(w.Body.String(), ShouldEqual, "Usage of the "+
						ruleDirection(rule)+" rule '"+rule.Name+"' is now restricted.\n")
				})

				Convey("Then the new access should be inserted "+
					"in the database", func() {

					var res []model.RuleAccess
					So(db.Select(&res, nil), ShouldBeNil)
					So(len(res), ShouldEqual, 1)
					So(res[0], ShouldResemble, expectedResult)
				})
			})
		}

		Convey("Given a server", func() {
			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)

			exp := model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   server.ID,
				ObjectType: server.TableName(),
			}

			handler := authorizeLocalAgent(logger, db)

			test(handler, "local_agent", server.Name, exp)
		})

		Convey("Given a partner", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(partner), ShouldBeNil)

			exp := model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   partner.ID,
				ObjectType: partner.TableName(),
			}

			handler := authorizeRemoteAgent(logger, db)

			test(handler, "remote_agent", partner.Name, exp)
		})
	})
}
