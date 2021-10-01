package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const testServersURI = "http://localhost:8080/api/servers/"

func TestListServers(t *testing.T) {
	logger := log.NewLogger("rest_server_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutServer) {
		Convey("Then the response body should contain an array "+
			"of the requested agents in JSON format", func() {
			exp, err := json.Marshal(expected)

			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})

		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})
	}

	Convey("Given the servers listing handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := listServers(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]OutServer{}

		Convey("Given a database with 4 servers", func() {
			a1 := model.LocalAgent{
				Name:        "server1",
				Protocol:    testProto1,
				Root:        "/root1",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			a2 := model.LocalAgent{
				Name:        "server2",
				Protocol:    testProto1,
				Root:        "/root2",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			a3 := model.LocalAgent{
				Name:        "server3",
				Protocol:    testProto1,
				Root:        "/root3",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:3",
			}
			a4 := model.LocalAgent{
				Name:        "server4",
				Protocol:    testProto2,
				Root:        "/root4",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:4",
			}

			So(db.Insert(&a1).Run(), ShouldBeNil)
			So(db.Insert(&a2).Run(), ShouldBeNil)
			So(db.Insert(&a3).Run(), ShouldBeNil)
			So(db.Insert(&a4).Run(), ShouldBeNil)

			agent1 := *FromLocalAgent(&a1, &AuthorizedRules{})
			agent2 := *FromLocalAgent(&a2, &AuthorizedRules{})
			agent3 := *FromLocalAgent(&a3, &AuthorizedRules{})
			agent4 := *FromLocalAgent(&a4, &AuthorizedRules{})

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
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol="+testProto1, nil)
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

func TestGetServer(t *testing.T) {
	logger := log.NewLogger("rest_server_get_test")

	Convey("Given the server get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			existing := model.LocalAgent{
				Name:        "existing",
				Protocol:    testProto1,
				Root:        "/root",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)

			Convey("Given a request with the valid server name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested server "+
						"in JSON format", func() {
						exp, err := json.Marshal(FromLocalAgent(&existing, &AuthorizedRules{}))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing server name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

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

func TestCreateServer(t *testing.T) {
	logger := log.NewLogger("rest_server_create_logger")

	Convey("Given the server creation handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := addServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			existing := model.LocalAgent{
				Name:        "existing",
				Protocol:    testProto1,
				Root:        "/root",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)

			Convey("Given a new server to insert in the database", func() {
				body := strings.NewReader(`{
					"name": "new_server",
					"protocol": "` + testProto1 + `",
					"root": "/new_root",
					"protoConfig": {},
					"address": "localhost:2"
				}`)

				Convey("Given that the new server is valid for insertion", func() {
					r, err := http.NewRequest(http.MethodPost, testServersURI, body)

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
							"of the new server", func() {
							location := w.Header().Get("Location")
							So(location, ShouldEqual, testServersURI+"new_server")
						})

						Convey("Then the new server should be inserted in "+
							"the database", func() {
							exp := model.LocalAgent{
								ID:          2,
								Owner:       database.Owner,
								Name:        "new_server",
								Protocol:    testProto1,
								Address:     "localhost:2",
								Root:        filepath.FromSlash("/new_root"),
								LocalInDir:  filepath.FromSlash("in"),
								LocalOutDir: filepath.FromSlash("out"),
								LocalTmpDir: filepath.FromSlash("tmp"),
								ProtoConfig: json.RawMessage("{}"),
							}
							var res model.LocalAgents
							So(db.Select(&res).Run(), ShouldBeNil)
							So(len(res), ShouldEqual, 2)
							So(res[1], ShouldResemble, exp)
						})

						Convey("Then the existing server should still be "+
							"present as well", func() {
							var rules model.LocalAgents
							So(db.Select(&rules).Run(), ShouldBeNil)
							So(len(rules), ShouldEqual, 2)

							So(rules[0], ShouldResemble, existing)
						})
					})
				})
			})
		})
	})
}

func TestDeleteServer(t *testing.T) {
	logger := log.NewLogger("rest_server_delete_test")

	Convey("Given the server deletion handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := deleteServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			existing := model.LocalAgent{
				Name:        "existing1",
				Protocol:    testProto1,
				Root:        "/root",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)

			Convey("Given a request with the valid agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, testServersURI+existing.Name, nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": existing.Name})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the agent should no longer be present in the database", func() {
						var agents model.LocalAgents
						So(db.Select(&agents).Run(), ShouldBeNil)
						So(agents, ShouldBeEmpty)
					})
				})
			})

			Convey("Given a request with a non-existing agent name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

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

func TestUpdateServer(t *testing.T) {
	logger := log.NewLogger("rest_server_update_logger")

	Convey("Given the agent updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := updateServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			old := model.LocalAgent{
				Name:        "old",
				Protocol:    testProto1,
				Address:     "localhost:1",
				Root:        "/old/root",
				LocalInDir:  "/old/in",
				LocalOutDir: "/old/out",
				LocalTmpDir: "/old/tmp",
				ProtoConfig: json.RawMessage(`{}`),
			}
			So(db.Insert(&old).Run(), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				body := strings.NewReader(`{
					"name": "update",
					"root": "/upt/root",
					"serverLocalInDir": "/upt/in",
					"serverLocalOutDir": "",
					"address": "localhost:2"
				}`)

				Convey("Given a valid name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, testServersURI+old.Name, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": old.Name})

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
						exp := model.LocalAgent{
							ID:         old.ID,
							Owner:      database.Owner,
							Name:       "update",
							Protocol:   testProto1,
							Address:    "localhost:2",
							Root:       filepath.FromSlash("/upt/root"),
							LocalInDir: filepath.FromSlash("/upt/in"),
							// sub-dirs cannot be empty if root isn't empty, so OutDir is reset to default
							LocalOutDir: filepath.FromSlash("out"),
							LocalTmpDir: filepath.FromSlash("/old/tmp"),
							ProtoConfig: json.RawMessage(`{}`),
						}

						var res model.LocalAgents
						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 1)

						So(res[0], ShouldResemble, exp)
					})
				})

				Convey("Given an invalid agent name", func() {
					r, err := http.NewRequest(http.MethodPatch, testServersURI+"toto", body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": "toto"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the agent was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old agent should still exist", func() {
						var agents model.LocalAgents
						So(db.Select(&agents).Run(), ShouldBeNil)
						So(agents, ShouldHaveLength, 1)
						So(agents[0], ShouldResemble, old)
					})
				})
			})
		})
	})
}

func TestReplaceServer(t *testing.T) {
	logger := log.NewLogger("rest_agent_update_logger")

	Convey("Given the agent updating handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := replaceServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			old := model.LocalAgent{
				Name:        "old",
				Protocol:    testProto1,
				Address:     "localhost:1",
				Root:        "/old/root",
				LocalInDir:  "/old/in",
				LocalOutDir: "/old/out",
				LocalTmpDir: "/old/tmp",
				ProtoConfig: json.RawMessage(`{}`),
			}
			So(db.Insert(&old).Run(), ShouldBeNil)

			Convey("Given new values to update the agent with", func() {
				body := strings.NewReader(`{
					"name": "update",
					"protocol": "` + testProto2 + `",
					"address": "localhost:2",
					"root": "/upt/root",
					"serverLocalInDir": "/upt/in",
					"serverLocalOutDir": "",
					"protoConfig": {}
				}`)

				Convey("Given a valid name parameter", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPatch, testServersURI+old.Name, body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": old.Name})

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
						exp := model.LocalAgent{
							ID:         old.ID,
							Owner:      database.Owner,
							Name:       "update",
							Protocol:   testProto2,
							Address:    "localhost:2",
							Root:       filepath.FromSlash("/upt/root"),
							LocalInDir: filepath.FromSlash("/upt/in"),
							// sub-dirs cannot be empty if root isn't empty, so OutDir is reset to default
							LocalOutDir: filepath.FromSlash("out"),
							LocalTmpDir: filepath.FromSlash("tmp"), // idem
							ProtoConfig: json.RawMessage(`{}`),
						}

						var res model.LocalAgents
						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 1)

						So(res[0], ShouldResemble, exp)
					})
				})

				Convey("Given an invalid agent name", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPatch, testServersURI+"toto", body)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"server": "toto"})

					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'NotFound'", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})

					Convey("Then the response body should state that "+
						"the agent was not found", func() {
						So(w.Body.String(), ShouldEqual, "server 'toto' not found\n")
					})

					Convey("Then the old agent should still exist", func() {
						var res model.LocalAgents
						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 1)
						So(res[0], ShouldResemble, old)
					})
				})
			})
		})
	})
}
