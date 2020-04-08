package rest

import (
	"bytes"
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

const remoteAgentsURI = "http://localhost:8080/api/partners/"

func TestListRemoteAgents(t *testing.T) {
	logger := log.NewLogger("rest_remote agent_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutRemoteAgent) {
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

	Convey("Given the remote agents listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listRemoteAgents(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutRemoteAgent{}

		Convey("Given a database with 4 remote agents", func() {
			a1 := &model.RemoteAgent{
				Name:        "remote agent1",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			a2 := &model.RemoteAgent{
				Name:        "remote agent2",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			a3 := &model.RemoteAgent{
				Name:        "remote agent3",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			a4 := &model.RemoteAgent{
				Name:        "remote agent4",
				Protocol:    "test2",
				ProtoConfig: []byte(`{}`),
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			agent1 := *FromRemoteAgent(a1, &AuthorizedRules{})
			agent2 := *FromRemoteAgent(a2, &AuthorizedRules{})
			agent3 := *FromRemoteAgent(a3, &AuthorizedRules{})
			agent4 := *FromRemoteAgent(a4, &AuthorizedRules{})

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []OutRemoteAgent{agent1, agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []OutRemoteAgent{agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []OutRemoteAgent{agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []OutRemoteAgent{agent4, agent3, agent2, agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with protocol parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol=test", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["partners"] = []OutRemoteAgent{agent1, agent2, agent3}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote agent_get_test")

	Convey("Given the remote agent get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 remote agent", func() {
			existing := &model.RemoteAgent{
				Name:        "existing",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with a valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested remote agent "+
						"in JSON format", func() {

						exp, err := json.Marshal(FromRemoteAgent(existing, &AuthorizedRules{}))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing remote agent name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto"})

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

func TestCreateRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote agent_create_logger")

	Convey("Given the remote agent creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 remote agent", func() {
			existing := &model.RemoteAgent{
				Name:        "existing",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a new remote agent to insert in the database", func() {
				newAgent := &InRemoteAgent{
					Name:        "new remote agent",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
				}

				Convey("Given that the new remote agent is valid for insertion", func() {
					body, err := json.Marshal(newAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAgentsURI,
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
							"of the new remote agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAgentsURI+newAgent.Name)
						})

						Convey("Then the new remote agent should be inserted in "+
							"the database", func() {
							exist, err := db.Exists(newAgent.ToModel())

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing remote agent should still be "+
							"present as well", func() {
							exist, err := db.Exists(existing)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}

func TestDeleteRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote agent_delete_test")

	Convey("Given the remote agent deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 remote agent", func() {
			existing := &model.RemoteAgent{
				Name:        "existing",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with a valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the agent should no longer be present in the database", func() {
						exist, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto"})

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

func TestUpdateRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_agent_update_logger")

	Convey("Given the agent updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 agents", func() {
			old := &model.RemoteAgent{
				Name:        "old",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			other := &model.RemoteAgent{
				Name:        "other",
				Protocol:    "test2",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				update := InRemoteAgent{
					Name:        "update",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{"key":"val"}`),
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

				Convey("Given a valid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAgentsURI+
						old.Name, bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": old.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, remoteAgentsURI+update.Name)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the agent should have been updated", func() {
							result := &model.RemoteAgent{ID: old.ID}
							So(db.Get(result), ShouldBeNil)

							So(result.Name, ShouldEqual, update.Name)
							So(result.Protocol, ShouldEqual, update.Protocol)

							protoConfig, err := json.Marshal(&update.ProtoConfig)
							So(err, ShouldBeNil)
							So(string(result.ProtoConfig), ShouldEqual, string(protoConfig))
						})
					})
				})

				Convey("Given an invalid agent name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remoteAgentsURI+"toto",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": "toto"})

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
							exist, err := db.Exists(old)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
