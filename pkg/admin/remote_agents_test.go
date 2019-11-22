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

const remoteAgentURI = APIPath + RemoteAgentsPath + "/"

func TestListRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote_agent_list_test", logConf)

	Convey("Given the remote agents listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listRemoteAgents(logger, db)

		remoteAgent1 := model.RemoteAgent{
			Name:        "remoteAgent1",
			Protocol:    "sftp",
			ProtoConfig: []byte("{}"),
		}
		remoteAgent2 := model.RemoteAgent{
			Name:        "remoteAgent2",
			Protocol:    "sftp",
			ProtoConfig: []byte("{}"),
		}
		remoteAgent3 := model.RemoteAgent{
			Name:        "remoteAgent3",
			Protocol:    "sftp",
			ProtoConfig: []byte("{}"),
		}
		remoteAgent4 := model.RemoteAgent{
			Name:        "remoteAgent4",
			Protocol:    "sftp",
			ProtoConfig: []byte("{}"),
		}

		agentListTest(handler, db, "remoteAgents", &remoteAgent1, &remoteAgent2,
			&remoteAgent3, &remoteAgent4)
	})
}

func TestGetRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote_agent_get_test", logConf)

	Convey("Given the remote agent get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 remote_agent", func() {
			expected := model.RemoteAgent{
				Name:        "existing",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid remote_agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAgentURI+id, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested remote_agent "+
						"in JSON format", func() {

						res := model.RemoteAgent{}
						err := json.Unmarshal(w.Body.Bytes(), &res)

						So(err, ShouldBeNil)
						So(res, ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing remote_agent ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, remoteAgentURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": "1000"})

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
	logger := log.NewLogger("rest_remote_agent_create_logger", logConf)

	Convey("Given the remote agent creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 remote_agent", func() {
			existingRemoteAgent := model.RemoteAgent{
				Name:        "existing",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&existingRemoteAgent)
			So(err, ShouldBeNil)

			Convey("Given a new remote agent to insert in the database", func() {
				newRemoteAgent := model.RemoteAgent{
					Name:        "new_remote_agent",
					Protocol:    "sftp",
					ProtoConfig: []byte("{\"new\":\"test\"}"),
				}

				Convey("Given that the new remote agent is valid for insertion", func() {
					body, err := json.Marshal(newRemoteAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new remote agent", func() {

							location := w.Header().Get("Location")
							So(location, ShouldStartWith, remoteAgentURI)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new remote agent should be inserted in the database", func() {
							exist, err := db.Exists(&newRemoteAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing remote_agent should still be present as well", func() {
							exist, err := db.Exists(&existingRemoteAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new remote agent has an ID", func() {
					newRemoteAgent.ID = existingRemoteAgent.ID

					body, err := json.Marshal(newRemoteAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the ID cannot be entered manually", func() {

							So(w.Body.String(), ShouldEqual, "The agent's ID "+
								"cannot be entered manually\n")
						})

						Convey("Then the new remote agent should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newRemoteAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new remote agent's name already exist", func() {
					newRemoteAgent.Name = existingRemoteAgent.Name

					body, err := json.Marshal(newRemoteAgent)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remoteAgentURI, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the name already exist", func() {

							So(w.Body.String(), ShouldEqual, "A remote agent with "+
								"the same name '"+newRemoteAgent.Name+"' already exist\n")
						})

						Convey("Then the new remote agent should NOT be inserted in the database", func() {
							exist, err := db.Exists(&newRemoteAgent)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestDeleteRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote_agent_delete_test", logConf)

	Convey("Given the remote agent deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteRemoteAgent(logger, db)

		Convey("Given a database with 1 remote_agent", func() {
			existing := model.RemoteAgent{
				Name:        "existing",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			agentDeleteTest(handler, db, "remote_agent", id, &existing)
		})
	})
}

func TestUpdateRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote_agent_update_logger", logConf)

	Convey("Given the remote agent updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 remote agents", func() {
			old := model.RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			other := model.RemoteAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the remote agent with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.RemoteAgent{
						ID:          old.ID,
						Name:        update.Name,
						Protocol:    old.Protocol,
						ProtoConfig: old.ProtoConfig,
					}

					checkValidUpdate(db, w, http.MethodPatch, remoteAgentURI,
						id, "remote_agent", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A remote agent with the same name '" + update.Name +
						"' already exist\n"
					checkInvalidUpdate(db, handler, w, body, remoteAgentURI, id,
						"remote_agent", &old, msg)
				})

				Convey("Given an invalid remote_agent ID parameter", func() {
					update := struct{}{}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, remoteAgentURI+"1000",
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"remote_agent": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Not Found' error", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})
					})
				})
			})
		})
	})
}

func TestReplaceRemoteAgent(t *testing.T) {
	logger := log.NewLogger("rest_remote_agent_replace_logger", logConf)

	Convey("Given the remote agent replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateRemoteAgent(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 remote agents", func() {
			old := model.RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			other := model.RemoteAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new remote agent", func() {
				replace := struct {
					Name, Protocol string
					ProtoConfig    []byte
				}{
					Name:        "replace",
					Protocol:    "sftp",
					ProtoConfig: []byte("{\"update\":\"test\"}"),
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.RemoteAgent{
					ID:          old.ID,
					Name:        replace.Name,
					Protocol:    replace.Protocol,
					ProtoConfig: replace.ProtoConfig,
				}

				checkValidUpdate(db, w, http.MethodPut, remoteAgentURI,
					id, "remote_agent", body, handler, &old, &expected)
			})

			Convey("Given a non-existing remote_agent ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, remoteAgentURI+"1000",
						bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": "1000"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}
