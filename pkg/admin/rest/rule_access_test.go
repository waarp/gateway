package rest

import (
	"encoding/json"
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

	Convey("Given a database with 1 rule", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		rule := &model.Rule{
			Name:   "rule",
			IsSend: true,
			Path:   "/rule_path",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

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

				Convey("Then the response body should state that access "+
					"to the rule is now restricted", func() {
					So(w.Body.String(), ShouldEqual, "Usage of the "+
						ruleDirection(rule)+" rule '"+rule.Name+"' is now restricted.")
				})

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the new access should be inserted "+
					"in the database", func() {
					var res model.RuleAccesses
					So(db.Select(&res).Run(), ShouldBeNil)
					So(len(res), ShouldEqual, 1)
					So(res[0], ShouldResemble, expectedResult)
				})
			})
		}

		Convey("Given a server", func() {
			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			exp := model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   server.ID,
				ObjectType: server.TableName(),
			}

			handler := authorizeServer(logger, db)
			vals["server"] = server.Name

			test(handler, exp)

			Convey("Given a local account", func() {
				account := &model.LocalAccount{
					LocalAgentID: server.ID,
					Login:        "toto",
					Password:     []byte("password"),
				}
				So(db.Insert(account).Run(), ShouldBeNil)

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
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			exp := model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   partner.ID,
				ObjectType: partner.TableName(),
			}

			handler := authorizePartner(logger, db)
			vals["partner"] = partner.Name

			test(handler, exp)

			Convey("Given a remote account", func() {
				account := &model.RemoteAccount{
					RemoteAgentID: partner.ID,
					Login:         "toto",
					Password:      []byte("password"),
				}
				So(db.Insert(account).Run(), ShouldBeNil)

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

	Convey("Given a database with 1 rule", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		rule := &model.Rule{
			Name:   "rule",
			IsSend: true,
			Path:   "/rule_path",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

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

				Convey("Then the response body should state that access to the rule "+
					"is now unrestricted", func() {
					So(w.Body.String(), ShouldEqual, "Usage of the "+ruleDirection(rule)+
						" rule '"+rule.Name+"' is now unrestricted.")
				})

				Convey("Then it should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the access should have been removed from the database", func() {
					var res model.RuleAccesses
					So(db.Select(&res).Run(), ShouldBeNil)
					So(len(res), ShouldEqual, 0)
				})
			})
		}

		Convey("Given a server", func() {
			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)
			vals["server"] = server.Name

			Convey("Given a server access", func() {
				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   server.ID,
					ObjectType: server.TableName(),
				}
				So(db.Insert(access).Run(), ShouldBeNil)

				handler := revokeServer(logger, db)
				test(handler)
			})

			Convey("Given a local account", func() {
				account := &model.LocalAccount{
					LocalAgentID: server.ID,
					Login:        "toto",
					Password:     []byte("password"),
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}
				So(db.Insert(access).Run(), ShouldBeNil)

				handler := revokeLocalAccount(logger, db)
				vals["local_account"] = account.Login

				test(handler)
			})
		})

		Convey("Given a partner", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)
			vals["partner"] = partner.Name

			Convey("Given a partner access", func() {
				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   partner.ID,
					ObjectType: partner.TableName(),
				}
				So(db.Insert(access).Run(), ShouldBeNil)

				handler := revokePartner(logger, db)

				test(handler)
			})

			Convey("Given a remote account", func() {
				account := &model.RemoteAccount{
					RemoteAgentID: partner.ID,
					Login:         "toto",
					Password:      []byte("password"),
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				access := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   account.ID,
					ObjectType: account.TableName(),
				}
				So(db.Insert(access).Run(), ShouldBeNil)

				handler := revokeRemoteAccount(logger, db)
				vals["remote_account"] = account.Login

				test(handler)
			})
		})
	})
}

func TestRuleAllowAll(t *testing.T) {
	logger := log.NewLogger("rest_revoke_rule_logger")

	Convey("Given a database with a rule", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		rule := &model.Rule{
			Name:   "rule",
			IsSend: true,
			Path:   "/rule_path",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given multiple accesses to that rule", func() {
			s := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			p := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(p).Run(), ShouldBeNil)
			So(db.Insert(s).Run(), ShouldBeNil)

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
			So(db.Insert(la).Run(), ShouldBeNil)
			So(db.Insert(ra).Run(), ShouldBeNil)

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
			So(db.Insert(sAcc).Run(), ShouldBeNil)
			So(db.Insert(pAcc).Run(), ShouldBeNil)
			So(db.Insert(laAcc).Run(), ShouldBeNil)
			So(db.Insert(raAcc).Run(), ShouldBeNil)

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
						var res model.RuleAccesses
						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 0)
					})
				})
			})
		})
	})
}
