package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const localAgentsURI = "http://localhost:8080" + APIPath + LocalAgentsPath + "/"

func TestListLocalAgents(t *testing.T) {
	logger := log.NewLogger("rest_local agent_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutAgent) {
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
		expected := map[string][]OutAgent{}

		Convey("Given a database with 4 local agents", func() {
			a1 := &model.LocalAgent{
				Name:        "local agent1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			a2 := &model.LocalAgent{
				Name:        "local agent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			a3 := &model.LocalAgent{
				Name:        "local agent3",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			a4 := &model.LocalAgent{
				Name:        "local agent4",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			agent1 := *FromLocalAgent(a1)
			agent2 := *FromLocalAgent(a2)
			agent3 := *FromLocalAgent(a3)
			agent4 := *FromLocalAgent(a4)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAgents"] = []OutAgent{agent1, agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAgents"] = []OutAgent{agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAgents"] = []OutAgent{agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAgents"] = []OutAgent{agent4, agent3, agent2, agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with protocol parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol=sftp", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["localAgents"] = []OutAgent{agent1, agent2, agent3, agent4}
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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(existing), ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid local agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAgentsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": id})

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

						exp, err := json.Marshal(FromLocalAgent(existing))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing local agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAgentsURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "1000"})

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a new local agent to insert in the database", func() {
				newAgent := &InAgent{
					Name:        "new local agent",
					Protocol:    "sftp",
					ProtoConfig: json.RawMessage(`{"address":"localhost","port":2023,"root":"/root"}`),
				}

				Convey("Given that the new local agent is valid for insertion", func() {
					body, err := json.Marshal(newAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAgentsURI, bytes.NewReader(body))

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
							So(location, ShouldStartWith, localAgentsURI)
						})

						Convey("Then the new local agent should be inserted in "+
							"the database", func() {
							exist, err := db.Exists(newAgent.ToLocal())

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing local agent should still be "+
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

func TestDeleteLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_delete_test")

	Convey("Given the local agent deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			existing := &model.LocalAgent{
				Name:        "existing1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(existing), ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, localAgentsURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": id})

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

			Convey("Given a request with a non-existing agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"local_agent": "1000"})

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			other := &model.LocalAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"titi"}`),
			}
			So(db.Create(old), ShouldBeNil)
			So(db.Create(other), ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the agent with", func() {

				Convey("Given a new login", func() {
					update := InAgent{
						Name:        "update",
						Protocol:    "sftp",
						ProtoConfig: json.RawMessage(`{"address":"localhost","port":2024,"root":"/root"}`),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, localAgentsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"local_agent": id})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, localAgentsURI+id)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the agent should have been updated", func() {
							result := &model.LocalAgent{ID: old.ID}
							err := db.Get(result)

							So(err, ShouldBeNil)
							So(result.Name, ShouldEqual, update.Name)
							So(result.Protocol, ShouldEqual, update.Protocol)

							protoConfig, err := json.Marshal(&update.ProtoConfig)
							So(err, ShouldBeNil)
							So(string(result.ProtoConfig), ShouldEqual, string(protoConfig))
						})
					})
				})

				Convey("Given an invalid agent ID", func() {
					update := InAgent{
						Name:        "update",
						Protocol:    "sftp",
						ProtoConfig: json.RawMessage(`{"address":"localhost","port":2024,"root":"/root"}`),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, localAgentsURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"local_agent": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the agent was not found", func() {
							So(w.Body.String(), ShouldEqual, "Record not found\n")
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
