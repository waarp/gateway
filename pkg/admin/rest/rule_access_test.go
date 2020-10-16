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
			Name:   "rule",
			IsSend: true,
			Path:   "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodPut, "", nil)
		So(err, ShouldBeNil)

		vals := map[string]string{
			"rule":      rule.Name,
			"direction": ruleDirection(rule),
		}

		test := func(handler http.Handler, expectedResult model.RuleAccess) {
			Convey("When sending the request to the handler", func() {
				r = mux.SetURLVars(r, vals)
				handler.ServeHTTP(w, r)

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the response body should state that access "+
					"to the rule is now restricted", func() {
					So(w.Body.String(), ShouldEqual, "Usage of the "+
						ruleDirection(rule)+" rule '"+rule.Name+"' is now restricted.")
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
			vals["local_agent"] = server.Name

			test(handler, exp)

			Convey("Given a local account", func() {
				account := &model.LocalAccount{
					LocalAgentID: server.ID,
					Login:        "toto",
					Password:     []byte("password"),
				}
				So(db.Create(account), ShouldBeNil)

				exp := model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}

				handler := authorizeLocalAccount(logger, db)
				vals["local_account"] = account.Login

				test(handler, exp)
			})
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
			vals["remote_agent"] = partner.Name

			test(handler, exp)

			Convey("Given a remote account", func() {
				account := &model.RemoteAccount{
					RemoteAgentID: partner.ID,
					Login:         "toto",
					Password:      []byte("password"),
				}
				So(db.Create(account), ShouldBeNil)

				exp := model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}

				handler := authorizeRemoteAccount(logger, db)
				vals["remote_account"] = account.Login

				test(handler, exp)
			})
		})
	})
}

func TestRevokeRule(t *testing.T) {
	logger := log.NewLogger("rest_revoke_rule_logger")

	Convey("Given a database with 1 rule", t, func() {
		db := database.GetTestDatabase()
		rule := &model.Rule{
			Name:   "rule",
			IsSend: true,
			Path:   "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodPut, "", nil)
		So(err, ShouldBeNil)

		vals := map[string]string{
			"rule":      rule.Name,
			"direction": ruleDirection(rule),
		}

		test := func(handler http.Handler) {
			Convey("When sending the request to the handler", func() {
				r = mux.SetURLVars(r, vals)
				handler.ServeHTTP(w, r)

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the response body should state that access to the rule "+
					"is now unrestricted", func() {
					So(w.Body.String(), ShouldEqual, "Usage of the "+ruleDirection(rule)+
						" rule '"+rule.Name+"' is now unrestricted.")
				})

				Convey("Then the access should have been removed from the database", func() {
					var res []model.RuleAccess
					So(db.Select(&res, nil), ShouldBeNil)
					So(len(res), ShouldEqual, 0)
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
			vals["local_agent"] = server.Name

			Convey("Given a server access", func() {
				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   server.ID,
					ObjectType: server.TableName(),
				}
				So(db.Create(access), ShouldBeNil)

				handler := revokeLocalAgent(logger, db)
				test(handler)
			})

			Convey("Given a local account", func() {
				account := &model.LocalAccount{
					LocalAgentID: server.ID,
					Login:        "toto",
					Password:     []byte("password"),
				}
				So(db.Create(account), ShouldBeNil)

				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}
				So(db.Create(access), ShouldBeNil)

				handler := revokeLocalAccount(logger, db)
				vals["local_account"] = account.Login

				test(handler)
			})
		})

		Convey("Given a partner", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(partner), ShouldBeNil)
			vals["remote_agent"] = partner.Name

			Convey("Given a partner access", func() {
				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   partner.ID,
					ObjectType: partner.TableName(),
				}
				So(db.Create(access), ShouldBeNil)

				handler := revokeRemoteAgent(logger, db)

				test(handler)
			})

			Convey("Given a remote account", func() {
				account := &model.RemoteAccount{
					RemoteAgentID: partner.ID,
					Login:         "toto",
					Password:      []byte("password"),
				}
				So(db.Create(account), ShouldBeNil)

				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}
				So(db.Create(access), ShouldBeNil)

				handler := revokeRemoteAccount(logger, db)
				vals["remote_account"] = account.Login

				test(handler)
			})
		})
	})
}

func TestRuleAllowAll(t *testing.T) {
	logger := log.NewLogger("rest_revoke_rule_logger")

	Convey("Given a database with a rule", t, func() {
		db := database.GetTestDatabase()
		rule := &model.Rule{
			Name:   "rule",
			IsSend: true,
			Path:   "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given multiple accesses to that rule", func() {
			s := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: []byte("{}"),
			}
			p := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: []byte("{}"),
			}
			So(db.Create(p), ShouldBeNil)
			So(db.Create(s), ShouldBeNil)

			la := &model.LocalAccount{
				LocalAgentID: s.ID,
				Login:        "toto",
				Password:     []byte("password"),
			}
			ra := &model.RemoteAccount{
				RemoteAgentID: p.ID,
				Login:         "tata",
				Password:      []byte("password"),
			}
			So(db.Create(la), ShouldBeNil)
			So(db.Create(ra), ShouldBeNil)

			sAcc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   s.ID,
				ObjectType: s.TableName(),
			}
			pAcc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   p.ID,
				ObjectType: p.TableName(),
			}
			laAcc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   la.ID,
				ObjectType: la.TableName(),
			}
			raAcc := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   ra.ID,
				ObjectType: ra.TableName(),
			}
			So(db.Create(sAcc), ShouldBeNil)
			So(db.Create(pAcc), ShouldBeNil)
			So(db.Create(laAcc), ShouldBeNil)
			So(db.Create(raAcc), ShouldBeNil)

			Convey("Given the 'allow_all' rule handler", func() {
				handler := allowAllRule(logger, db)

				Convey("When sending a request to the handler", func() {
					w := httptest.NewRecorder()
					r, err := http.NewRequest(http.MethodPut, "", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{
						"rule":      rule.Name,
						"direction": ruleDirection(rule),
					})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the response body should state that access to the rule "+
						"is now unrestricted", func() {
						So(w.Body.String(), ShouldEqual, "Usage of the "+ruleDirection(rule)+
							" rule '"+rule.Name+"' is now unrestricted.")
					})

					Convey("Then all accesses should have been removed from the database", func() {
						var res []model.RuleAccess
						So(db.Select(&res, nil), ShouldBeNil)
						So(len(res), ShouldEqual, 0)
					})
				})
			})
		})
	})
}
