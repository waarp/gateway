package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const localAgentsURI = "http://localhost:8080/api/servers/"

func TestListLocalAgents(t *testing.T) {
	logger := log.NewLogger("rest_local agent_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutServer) {
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

	Convey("Given the local agents listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listLocalAgents(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutServer{}

		Convey("Given a database with 4 local agents", func() {
			a1 := &model.LocalAgent{
				Name:        "local agent1",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root1"},
				ProtoConfig: []byte(`{}`),
			}
			a2 := &model.LocalAgent{
				Name:        "local agent2",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root2"},
				ProtoConfig: []byte(`{}`),
			}
			a3 := &model.LocalAgent{
				Name:        "local agent3",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root3"},
				ProtoConfig: []byte(`{}`),
			}
			a4 := &model.LocalAgent{
				Name:        "local agent4",
				Protocol:    "test2",
				Paths:       &model.ServerPaths{Root: "/root4"},
				ProtoConfig: []byte(`{}`),
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			agent1 := *FromLocalAgent(a1, &AuthorizedRules{})
			agent2 := *FromLocalAgent(a2, &AuthorizedRules{})
			agent3 := *FromLocalAgent(a3, &AuthorizedRules{})
			agent4 := *FromLocalAgent(a4, &AuthorizedRules{})

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []OutServer{agent1, agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []OutServer{agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []OutServer{agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []OutServer{agent4, agent3, agent2, agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with protocol parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol=test", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []OutServer{agent1, agent2, agent3}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_get_test")

	Convey("Given the local agent get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			existing := &model.LocalAgent{
				Name:        "existing",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root"},
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with the valid local agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested local agent "+
						"in JSON format", func() {

						exp, err := json.Marshal(FromLocalAgent(existing, &AuthorizedRules{}))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing local agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "toto"})

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

func TestCreateLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_create_logger")

	Convey("Given the local agent creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			existing := &model.LocalAgent{
				Name:        "existing",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root"},
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a new local agent to insert in the database", func() {
				newAgent := &InServer{
					Name:        "new local agent",
					Protocol:    "test",
					Paths:       &model.ServerPaths{Root: "/root"},
					ProtoConfig: json.RawMessage(`{}`),
				}

				Convey("Given that the new local agent is valid for insertion", func() {
					body, err := json.Marshal(newAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAgentsURI,
						bytes.NewReader(body))

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
							"of the new local agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, localAgentsURI+
								url.PathEscape(newAgent.Name))
						})

						Convey("Then the new local agent should be inserted in "+
							"the database", func() {
							exp := model.LocalAgent{
								ID:       2,
								Owner:    database.Owner,
								Name:     newAgent.Name,
								Protocol: newAgent.Protocol,
								Paths: &model.ServerPaths{
									Root:    "/root",
									InDir:   "in",
									OutDir:  "out",
									WorkDir: "work",
								},
								ProtoConfig: newAgent.ProtoConfig,
							}
							var res []model.LocalAgent
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 2)
							So(res[1], ShouldResemble, exp)
						})

						Convey("Then the existing local agent should still be "+
							"present as well", func() {
							var rules []model.LocalAgent
							So(db.Select(&rules, nil), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)

							So(rules[0], ShouldResemble, *existing)
						})
					})
				})
			})
		})
	})
}

func TestDeleteLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_delete_test")

	Convey("Given the local agent deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			existing := &model.LocalAgent{
				Name:        "existing1",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root"},
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with the valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, localAgentsURI+existing.Name, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the agent should no longer be present in the database", func() {
						var rules []model.LocalAgent
						So(db.Select(&rules, nil), ShouldBeNil)
						So(rules, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "toto"})

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

func TestUpdateLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_agent_update_logger")

	Convey("Given the agent updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 agents", func() {
			old := &model.LocalAgent{
				Name:        "old",
				Protocol:    "test",
				Paths:       &model.ServerPaths{Root: "/root"},
				ProtoConfig: []byte(`{}`),
			}
			other := &model.LocalAgent{
				Name:        "other",
				Protocol:    "test2",
				Paths:       &model.ServerPaths{Root: "/root"},
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				update := InServer{
					Name:        "update",
					Protocol:    "test",
					Paths:       &model.ServerPaths{Root: "/new_root"},
					ProtoConfig: json.RawMessage(`{"key":"val"}`),
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

				Convey("Given a valid name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, localAgentsURI+old.Name,
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": old.Name})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated agent", func() {

						location := w.Header().Get("Location")
						So(location, ShouldEqual, localAgentsURI+update.Name)
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the agent should have been updated", func() {
						result := &model.LocalAgent{ID: old.ID}

						So(db.Get(result), ShouldBeNil)
						So(result.Name, ShouldEqual, update.Name)
						So(result.Protocol, ShouldEqual, update.Protocol)

						protoConfig, err := json.Marshal(&update.ProtoConfig)
						So(err, ShouldBeNil)
						So(string(result.ProtoConfig), ShouldEqual, string(protoConfig))
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPatch, localAgentsURI+"toto",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "toto"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the agent was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old agent should still exist", func() {
						So(db.Get(old), ShouldBeNil)
					})
				})
			})
		})
	})
}
