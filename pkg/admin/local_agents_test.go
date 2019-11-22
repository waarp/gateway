package admin

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

const localAgentURI = APIPath + LocalAgentsPath + "/"

func TestListLocalAgents(t *testing.T) {
	logger := log.NewLogger("rest_local agent_list_test", logConf)

	Convey("Given the local agents listing handler", t, func() {
		db := database.GetTestDatabase()

		localAgent1 := model.LocalAgent{
			Name:        "local agent1",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		localAgent2 := model.LocalAgent{
			Name:        "local agent2",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		localAgent3 := model.LocalAgent{
			Name:        "local agent3",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		localAgent4 := model.LocalAgent{
			Name:        "local agent4",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		handler := listLocalAgents(logger, db)

		agentListTest(handler, db, "localAgents", &localAgent1, &localAgent2,
			&localAgent3, &localAgent4)
	})
}

func TestGetLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_get_test", logConf)

	Convey("Given the local agent get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			expected := model.LocalAgent{
				Name:        "existing",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid local agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAgentURI+id, nil)
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

						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing local agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, localAgentURI+"1000", nil)
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
	logger := log.NewLogger("rest_local agent_create_logger", logConf)

	Convey("Given the local agent creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 local agent", func() {
			existingLocalAgent := model.LocalAgent{
				Name:        "existing",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&existingLocalAgent)
			So(err, ShouldBeNil)

			Convey("Given a new local agent to insert in the database", func() {
				newLocalAgent := model.LocalAgent{
					Name:        "new_local agent",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"tata"}`),
				}

				Convey("Given that the new local agent is valid for insertion", func() {
					body, err := json.Marshal(newLocalAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new local agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldStartWith, localAgentURI)
						})

						Convey("Then the new local agent should be inserted in "+
							"the database", func() {
							exist, err := db.Exists(&newLocalAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing local agent should still be "+
							"present as well", func() {
							exist, err := db.Exists(&existingLocalAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})
					})
				})

				Convey("Given that the new local agent has an ID", func() {
					newLocalAgent.ID = existingLocalAgent.ID

					body, err := json.Marshal(newLocalAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message "+
							"stating that IDs cannot be entered manually", func() {

							So(w.Body.String(), ShouldEqual, "The agent's ID "+
								"cannot be entered manually\n")
						})

						Convey("Then the new local agent should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newLocalAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new local agent's name already exist", func() {
					newLocalAgent.Name = existingLocalAgent.Name

					body, err := json.Marshal(newLocalAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, localAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the name already exist", func() {

							So(w.Body.String(), ShouldEqual, "A local agent with "+
								"the same name '"+newLocalAgent.Name+"' already exist\n")
						})

						Convey("Then the new local agent should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newLocalAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestDeleteLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_delete_test", logConf)

	Convey("Given the local agent deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteLocalAgent(logger, db)

		Convey("Given a database with 1 local agent", func() {
			existing := model.LocalAgent{
				Name:        "existing1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			agentDeleteTest(handler, db, "local_agent", id, &existing)
		})
	})
}

func TestUpdateLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_update_logger", logConf)

	Convey("Given the local agent updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 local agents", func() {
			old := model.LocalAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			other := model.LocalAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the local agent with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.LocalAgent{
						Name:        update.Name,
						Protocol:    old.Protocol,
						ProtoConfig: old.ProtoConfig,
					}

					checkValidUpdate(db, w, http.MethodPatch, localAgentURI,
						id, "local_agent", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A local agent with the same name '" + update.Name +
						"' already exist\n"
					checkInvalidUpdate(db, handler, w, body, localAgentURI, id,
						"local_agent", &old, msg)
				})

				Convey("Given an invalid type", func() {
					update := struct{ Protocol string }{Protocol: "not a protocol"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "The agent's protocol must be one of: [sftp]\n"
					checkInvalidUpdate(db, handler, w, body, localAgentURI, id,
						"local_agent", &old, msg)
				})
			})
		})
	})
}

func TestReplaceLocalAgent(t *testing.T) {
	logger := log.NewLogger("rest_local agent_replace_logger", logConf)

	Convey("Given the local agent replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateLocalAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 local agents", func() {
			old := model.LocalAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			other := model.LocalAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new local agent", func() {
				replace := model.LocalAgent{
					Name:        "replace",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.LocalAgent{
					Name:        replace.Name,
					Protocol:    replace.Protocol,
					ProtoConfig: replace.ProtoConfig,
				}

				checkValidUpdate(db, w, http.MethodPut, localAgentURI,
					id, "local_agent", body, handler, &old, &expected)
			})

			Convey("Given a non-existing local agent ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, localAgentURI+"1000",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should say the entry was not found", func() {
						So(w.Body.String(), ShouldEqual, "Record not found\n")
					})
				})
			})
		})
	})
}
