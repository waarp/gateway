package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const testPartnersURI = "http://localhost:8080/api/partners/"

func TestListPartners(t *testing.T) {
	check := func(w *httptest.ResponseRecorder, expected map[string][]*OutPartner) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested agents in JSON format", func() {
			exp, err := json.Marshal(expected)

			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given the partners listing handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partners_list_test")
		db := database.TestDatabase(c)
		handler := listPartners(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]*OutPartner{}

		Convey("Given a database with 4 partners", func() {
			a1 := &model.RemoteAgent{
				Name: "partner1", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			a2 := &model.RemoteAgent{
				Name: "partner2", Protocol: testProto1,
				Address: types.Addr("localhost", 2),
			}
			a3 := &model.RemoteAgent{
				Name: "partner3", Protocol: testProto1,
				Address: types.Addr("localhost", 3),
			}
			a4 := &model.RemoteAgent{
				Name: "partner4", Protocol: testProto2,
				Address: types.Addr("localhost", 4),
			}

			So(db.Insert(a1).Run(), ShouldBeNil)
			So(db.Insert(a2).Run(), ShouldBeNil)
			So(db.Insert(a3).Run(), ShouldBeNil)
			So(db.Insert(a4).Run(), ShouldBeNil)

			agent1, err := DBPartnerToREST(db, a1)
			So(err, ShouldBeNil)
			agent2, err := DBPartnerToREST(db, a2)
			So(err, ShouldBeNil)
			agent3, err := DBPartnerToREST(db, a3)
			So(err, ShouldBeNil)
			agent4, err := DBPartnerToREST(db, a4)
			So(err, ShouldBeNil)

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []*OutPartner{agent1, agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []*OutPartner{agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []*OutPartner{agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []*OutPartner{agent4, agent3, agent2, agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with protocol parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol="+testProto1, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []*OutPartner{agent1, agent2, agent3}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetPartner(t *testing.T) {
	Convey("Given the partner get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partner_get_test")
		db := database.TestDatabase(c)
		handler := getPartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			existing := &model.RemoteAgent{
				Name: "existing", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			pswd := model.Credential{
				RemoteAgentID: utils.NewNullInt64(existing.ID),
				Name:          "partner pswd",
				Type:          auth.Password,
				Value:         "sesame",
			}
			So(db.Insert(&pswd).Run(), ShouldBeNil)

			rule := model.Rule{Name: "rule name", IsSend: true}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			Convey("Given a request with a valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested partner "+
						"in JSON format", func() {
						So(w.Body.String(), ShouldResemble, `{`+
							`"name":"`+existing.Name+`",`+
							`"protocol":"`+existing.Protocol+`",`+
							`"address":"`+existing.Address.String()+`",`+
							`"credentials":["`+pswd.Name+`"],`+
							`"protoConfig":{},`+
							`"authorizedRules":{"sending":["`+rule.Name+`"]}`+
							"}\n")
					})
				})
			})

			Convey("Given a request with a non-existing partner name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestCreatePartner(t *testing.T) {
	Convey("Given the partner creation handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partner_create_logger")
		db := database.TestDatabase(c)
		handler := addPartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			existing := &model.RemoteAgent{
				Name: "existing", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a new partner to insert in the database", func() {
				body := strings.NewReader(`{
					"name": "new_partner",
					"protocol": "` + testProto1 + `",
					"protoConfig": {},
					"address": "localhost:2"
				}`)

				Convey("Given that the new partner is valid for insertion", func() {
					r, err := http.NewRequest(http.MethodPost, testPartnersURI, body)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new partner", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, testPartnersURI+"new_partner")
						})

						Convey("Then the new partner should be inserted in "+
							"the database", func() {
							var ags model.RemoteAgents
							So(db.Select(&ags).Run(), ShouldBeNil)
							So(len(ags), ShouldEqual, 2)

							So(ags[1], ShouldResemble, &model.RemoteAgent{
								ID:          2,
								Owner:       conf.GlobalConfig.GatewayName,
								Name:        "new_partner",
								Protocol:    testProto1,
								Address:     types.Addr("localhost", 2),
								ProtoConfig: map[string]any{},
							})
						})

						Convey("Then the existing partner should still be "+
							"present as well", func() {
							var ags model.RemoteAgents
							So(db.Select(&ags).Run(), ShouldBeNil)
							So(len(ags), ShouldEqual, 2)

							So(ags[0], ShouldResemble, existing)
						})
					})
				})
			})
		})
	})
}

func TestDeletePartner(t *testing.T) {
	Convey("Given the partner deletion handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partner_delete_test")
		db := database.TestDatabase(c)
		handler := deletePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner", func() {
			existing := model.RemoteAgent{
				Name: "existing", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)

			Convey("Given a request with a valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the agent should no longer be present in the database", func() {
						var ags model.RemoteAgents
						So(db.Select(&ags).Run(), ShouldBeNil)
						So(ags, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestUpdatePartner(t *testing.T) {
	Convey("Given the agent updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partner_update_logger")
		db := database.TestDatabase(c)
		handler := updatePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			old := &model.RemoteAgent{
				Name: "old", Protocol: testProto1,
				Address:     types.Addr("localhost", 1),
				ProtoConfig: map[string]any{"old_key": "old_val"},
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				const (
					newName  = "update"
					newProto = testProto2
					newAddr  = "localhost:2"
				)
				newConf := map[string]any{"new_key": "new_val"}

				body := map[string]any{
					"name":        newName,
					"protocol":    newProto,
					"protoConfig": newConf,
					"address":     newAddr,
				}

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, testPartnersURI+
						old.Name, utils.ToJSONBody(body))
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": old.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated agent", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, testPartnersURI+"update")
						})

						Convey("Then the agent should have been updated", func() {
							exp := &model.RemoteAgent{
								ID:          old.ID,
								Owner:       conf.GlobalConfig.GatewayName,
								Name:        newName,
								Protocol:    newProto,
								Address:     mustAddr(newAddr),
								ProtoConfig: newConf,
							}

							var parts model.RemoteAgents
							So(db.Select(&parts).Run(), ShouldBeNil)
							So(len(parts), ShouldEqual, 1)

							So(parts[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, testPartnersURI+"toto",
						utils.ToJSONBody(body))
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the agent was not found", func() {
							So(w.Body.String(), ShouldEqual, "partner 'toto' not found\n")
						})

						Convey("Then the old agent should still exist", func() {
							var ags model.RemoteAgents
							So(db.Select(&ags).Run(), ShouldBeNil)
							So(len(ags), ShouldEqual, 1)

							So(ags[0], ShouldResemble, old)
						})
					})
				})
			})
		})
	})
}

func TestReplacePartner(t *testing.T) {
	Convey("Given the agent updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_partner_update_logger")
		db := database.TestDatabase(c)
		handler := replacePartner(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 agents", func() {
			old := &model.RemoteAgent{
				Name: "old", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				body := strings.NewReader(`{
					"name": "update",
					"protocol": "` + testProto2 + `",
					"protoConfig": {},
					"address": "localhost:2"
				}`)

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, testServersURI+
						old.Name, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": old.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated agent", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, testServersURI+"update")
						})

						Convey("Then the agent should have been updated", func() {
							exp := &model.RemoteAgent{
								ID:          old.ID,
								Owner:       conf.GlobalConfig.GatewayName,
								Name:        "update",
								Protocol:    testProto2,
								Address:     types.Addr("localhost", 2),
								ProtoConfig: map[string]any{},
							}

							var parts model.RemoteAgents
							So(db.Select(&parts).Run(), ShouldBeNil)
							So(len(parts), ShouldEqual, 1)

							So(parts[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPut, testServersURI+"toto", body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"partner": "toto"})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the agent was not found", func() {
							So(w.Body.String(), ShouldEqual, "partner 'toto' not found\n")
						})

						Convey("Then the old agent should still exist", func() {
							var ags model.RemoteAgents
							So(db.Select(&ags).Run(), ShouldBeNil)
							So(len(ags), ShouldEqual, 1)

							So(ags[0], ShouldResemble, old)
						})
					})
				})
			})
		})
	})
}
