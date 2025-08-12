package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const testServersURI = "http://localhost:8080/api/servers/"

func TestListServers(t *testing.T) {
	check := func(w *httptest.ResponseRecorder, expected map[string][]*OutServer) {
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
		logger := testhelpers.TestLogger(c, "rest_server_list_test")
		db := database.TestDatabase(c)
		handler := listServers(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]*OutServer{}

		Convey("Given a database with 4 servers", func() {
			a1 := &model.LocalAgent{
				Name: "server1", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			a2 := &model.LocalAgent{
				Name: "server2", Protocol: testProto1,
				Address: types.Addr("localhost", 2),
			}
			a3 := &model.LocalAgent{
				Name: "server3", Protocol: testProto1,
				Address: types.Addr("localhost", 3),
			}
			a4 := &model.LocalAgent{
				Name: "server4", Protocol: testProto2,
				Address: types.Addr("localhost", 4),
			}

			So(db.Insert(a1).Run(), ShouldBeNil)
			So(db.Insert(a2).Run(), ShouldBeNil)
			So(db.Insert(a3).Run(), ShouldBeNil)
			So(db.Insert(a4).Run(), ShouldBeNil)

			agent1, err := DBServerToREST(db, a1)
			So(err, ShouldBeNil)
			agent2, err := DBServerToREST(db, a2)
			So(err, ShouldBeNil)
			agent3, err := DBServerToREST(db, a3)
			So(err, ShouldBeNil)
			agent4, err := DBServerToREST(db, a4)
			So(err, ShouldBeNil)

			// add a server from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			a5 := model.LocalAgent{
				Name: "server5", Protocol: testProto1,
				Address: types.Addr("localhost", 5),
			}
			So(db.Insert(&a5).Run(), ShouldBeNil)

			conf.GlobalConfig.GatewayName = owner

			Convey("Given a request with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []*OutServer{agent1, agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []*OutServer{agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []*OutServer{agent2, agent3, agent4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []*OutServer{agent4, agent3, agent2, agent1}
					check(w, expected)
				})
			})

			Convey("Given a request with protocol parameters", func() {
				r, err := http.NewRequest(http.MethodGet, "?type=http&protocol="+testProto1, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["servers"] = []*OutServer{agent1, agent2, agent3}
					check(w, expected)
				})
			})
		})
	})
}

func TestGetServer(t *testing.T) {
	Convey("Given the server get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_server_get_test")
		db := database.TestDatabase(c)
		handler := getServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			// add a server from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			other := &model.LocalAgent{
				Name: "existing", Protocol: testProto1,
				Address: types.Addr("localhost", 10),
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			conf.GlobalConfig.GatewayName = owner

			existing := &model.LocalAgent{
				Name: other.Name, Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			pswd := model.Credential{
				LocalAgentID: utils.NewNullInt64(existing.ID),
				Name:         "server password",
				Type:         auth.Password,
				Value:        "sesame",
			}
			So(db.Insert(&pswd).Run(), ShouldBeNil)

			rule := model.Rule{Name: "rule name", IsSend: false}
			So(db.Insert(&rule).Run(), ShouldBeNil)

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
						So(w.Body.String(), ShouldResemble, `{`+
							`"name":"`+existing.Name+`",`+
							`"protocol":"`+existing.Protocol+`",`+
							`"enabled":`+strconv.FormatBool(!existing.Disabled)+`,`+
							`"address":"`+existing.Address.String()+`",`+
							`"credentials":["`+pswd.Name+`"],`+
							`"protoConfig":{},`+
							`"authorizedRules":{"reception":["`+rule.Name+`"]}`+
							"}\n")
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
	Convey("Given the server creation handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_server_create_logger")
		db := database.TestDatabase(c)
		handler := addServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			existing := &model.LocalAgent{
				Name: "existing", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given a new server to insert in the database", func() {
				body := strings.NewReader(`{
					"name": "new_server",
					"protocol": "` + testProto1 + `",
					"rootDir": "/new_root",
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
							var res model.LocalAgents

							So(db.Select(&res).Run(), ShouldBeNil)
							So(len(res), ShouldEqual, 2)
							So(res[1], ShouldResemble, &model.LocalAgent{
								ID:            2,
								Owner:         conf.GlobalConfig.GatewayName,
								Name:          "new_server",
								Protocol:      testProto1,
								Address:       types.Addr("localhost", 2),
								RootDir:       "/new_root",
								ReceiveDir:    "in",
								SendDir:       "out",
								TmpReceiveDir: "tmp",
								ProtoConfig:   map[string]any{},
							})
						})

						Convey("Then it should have added (but not started) the server to the service list", func() {
							const name = "new_server"
							So(services.Servers, ShouldContainKey, name)
							So(stateCode(services.Servers[name]), ShouldEqual, utils.StateOffline)
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
	Convey("Given the server deletion handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_server_delete_test")
		db := database.TestDatabase(c)
		handler := deleteServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 server", func() {
			existing := model.LocalAgent{
				Name: "existing1", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(&existing).Run(), ShouldBeNil)

			protoService := &testService{}
			services.Servers[existing.Name] = protoService

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

					Convey("Then it should have stopped the service", func() {
						So(services.Servers, ShouldNotContainKey, existing.Name)
						So(stateCode(protoService), ShouldEqual, utils.StateOffline)
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

			Convey("Given that the service is running", func() {
				So(protoService.Start(), ShouldBeNil)

				r, err := http.NewRequest(http.MethodDelete, testServersURI+existing.Name, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"server": existing.Name})
				handler.ServeHTTP(w, r)

				Convey("Then it should reply with a 'No Content'", func() {
					So(w.Code, ShouldEqual, http.StatusNoContent)
				})

				Convey("Then it should have stopped the service", func() {
					code, _ := protoService.state.Get()
					So(code, ShouldEqual, utils.StateOffline)
				})

				Convey("Then it should have removed the server from the service list", func() {
					So(services.Servers, ShouldNotContainKey, existing.Name)
				})
			})
		})
	})
}

func TestUpdateServer(t *testing.T) {
	Convey("Given the agent updating handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_server_update_logger")
		db := database.TestDatabase(c)
		handler := updateServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			old := &model.LocalAgent{
				Name:          "old",
				Protocol:      testProto1,
				Address:       types.Addr("localhost", 1),
				RootDir:       "/old/root",
				ReceiveDir:    "/old/in",
				SendDir:       "/old/out",
				TmpReceiveDir: "/old/tmp",
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			protoService := makeAndStartTestService()
			services.Servers[old.Name] = protoService
			defer delete(services.Servers, old.Name)

			Convey("Given new values to update the agent with", func() {
				body := strings.NewReader(`{
					"name": "update",
					"rootDir": "/upt/root",
					"receiveDir": "/upt/in",
					"sendDir": "",
					"address": "localhost:2"
				}`)

				Convey("Given a valid name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, testServersURI+old.Name, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"server": old.Name})
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated agent", func() {
						location := w.Header().Get("Location")
						So(location, ShouldEqual, testServersURI+"update")
					})

					Convey("Then the agent should have been updated", func() {
						var res model.LocalAgents

						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 1)
						So(res[0], ShouldResemble, &model.LocalAgent{
							ID:         old.ID,
							Owner:      conf.GlobalConfig.GatewayName,
							Name:       "update",
							Protocol:   testProto1,
							Address:    types.Addr("localhost", 2),
							RootDir:    "/upt/root",
							ReceiveDir: "/upt/in",
							// sub-dirs cannot be empty if root isn't empty, so OutDir is reset to default
							SendDir:       "out",
							TmpReceiveDir: "/old/tmp",
							ProtoConfig:   map[string]any{},
						})
					})

					Convey("Then the service should have been restarted", func() {
						So(stateCode(protoService), ShouldEqual, utils.StateOffline)

						const newName = "update"

						So(services.Servers, ShouldNotContainKey, old.Name)
						So(services.Servers, ShouldContainKey, newName)
						So(stateCode(services.Servers[newName]), ShouldEqual, utils.StateRunning)
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
						So(w.Body.String(), ShouldEqual, "server \"toto\" not found\n")
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
	Convey("Given the agent replacing handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_agent_replace_logger")
		db := database.TestDatabase(c)
		handler := replaceServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			old := &model.LocalAgent{
				Name:          "old",
				Protocol:      testProto1,
				Address:       types.Addr("localhost", 1),
				RootDir:       "/old/root",
				ReceiveDir:    "/old/in",
				SendDir:       "/old/out",
				TmpReceiveDir: "/old/tmp",
			}
			So(db.Insert(old).Run(), ShouldBeNil)

			protoService := makeAndStartTestService()
			services.Servers[old.Name] = protoService
			defer delete(services.Servers, old.Name)

			Convey("Given new values to update the agent with", func() {
				body := strings.NewReader(`{
					"name": "update",
					"protocol": "` + testProto2 + `",
					"address": "localhost:2",
					"rootDir": "/upt/root",
					"receiveDir": "/upt/in",
					"sendDir": "",
					"protoConfig": {}
				}`)

				Convey("Given a valid name parameter", func() {
					//nolint:noctx // this is a test
					r, err := http.NewRequest(http.MethodPatch, testServersURI+old.Name, body)
					So(err, ShouldBeNil)

					r = mux.SetURLVars(r, map[string]string{"server": old.Name})
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'Created'", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain "+
						"the URI of the updated agent", func() {
						location := w.Header().Get("Location")
						So(location, ShouldEqual, testServersURI+"update")
					})

					Convey("Then the agent should have been updated", func() {
						var res model.LocalAgents

						So(db.Select(&res).Run(), ShouldBeNil)
						So(len(res), ShouldEqual, 1)

						So(res[0], ShouldResemble, &model.LocalAgent{
							ID:         old.ID,
							Owner:      conf.GlobalConfig.GatewayName,
							Name:       "update",
							Protocol:   testProto2,
							Address:    types.Addr("localhost", 2),
							RootDir:    "/upt/root",
							ReceiveDir: "/upt/in",
							// sub-dirs cannot be empty if root isn't empty, so OutDir is reset to default
							SendDir:       "out",
							TmpReceiveDir: "tmp", // idem
							ProtoConfig:   map[string]any{},
						})
					})

					Convey("Then the service should have been restarted", func() {
						So(stateCode(protoService), ShouldEqual, utils.StateOffline)

						const newName = "update"

						So(services.Servers, ShouldNotContainKey, old.Name)
						So(services.Servers, ShouldContainKey, newName)
						So(stateCode(services.Servers[newName]), ShouldEqual, utils.StateRunning)
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
						So(w.Body.String(), ShouldEqual, "server \"toto\" not found\n")
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

func TestEnableDisableServer(t *testing.T) {
	testEnableDisableServer := func(expectedDisabled bool) {
		path, name := "/servers/{server}/disable", "disable"
		if !expectedDisabled {
			path, name = "/servers/{server}/enable", "enable"
		}

		Convey("Given the agent "+name+" handler", t, func(c C) {
			logger := testhelpers.TestLogger(c, "rest_agent_"+name+"_logger")
			db := database.TestDatabase(c)
			host := testAdminServer(logger, db)

			Convey("Given a database with a "+name+"d agent", func() {
				agent := model.LocalAgent{
					Name: "agent", Protocol: testProto1,
					Disabled: !expectedDisabled,
					Address:  types.Addr("localhost", 1),
				}
				So(db.Insert(&agent).Run(), ShouldBeNil)

				path = strings.ReplaceAll(path, "{server}", agent.Name)

				Convey("When sending a request to the handler", func() {
					resp := methodTestRequest(host, path)

					Convey("Then it should reply 'ACCEPTED'", func() {
						So(resp.StatusCode, ShouldEqual, http.StatusAccepted)

						Convey("Then it should have "+name+"d the sever", func() {
							var check model.LocalAgent

							So(db.Get(&check, "id=?", agent.ID).Run(), ShouldBeNil)
							So(check.Disabled, ShouldEqual, expectedDisabled)
						})
					})
				})
			})
		})
	}

	testEnableDisableServer(true)
	testEnableDisableServer(false)
}

func TestStartServer(t *testing.T) {
	Convey("Given the server start handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_agent_update_logger")
		db := database.TestDatabase(c)
		handle := startServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			agent := model.LocalAgent{
				Name: "local server", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(&agent).Run(), ShouldBeNil)

			Convey("Given a valid name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/start", nil)
				r = mux.SetURLVars(r, map[string]string{"server": agent.Name})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 202 code", func() {
					So(w.Body.String(), ShouldBeEmpty)
					So(w.Code, ShouldEqual, http.StatusAccepted)
				})

				Convey("Then it should have started the service", func() {
					So(services.Servers, ShouldContainKey, agent.Name)
					So(stateCode(services.Servers[agent.Name]), ShouldEqual, utils.StateRunning)
				})
			})

			Convey("Given an incorrect name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/start", nil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 404 code", func() {
					So(w.Code, ShouldEqual, http.StatusNotFound)
					So(w.Body.String(), ShouldEqual, "server \"toto\" not found\n")
				})
			})
		})
	})
}

func TestStopServer(t *testing.T) {
	Convey("Given the server stop handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_agent_update_logger")
		db := database.TestDatabase(c)
		handle := stopServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			agent := model.LocalAgent{
				Name: "local server", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(&agent).Run(), ShouldBeNil)

			services.Servers[agent.Name] = makeAndStartTestService()
			defer delete(services.Servers, agent.Name)

			Convey("Given a valid name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/stop", nil)
				r = mux.SetURLVars(r, map[string]string{"server": agent.Name})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 202 code", func() {
					So(w.Body.String(), ShouldBeEmpty)
					So(w.Code, ShouldEqual, http.StatusAccepted)
				})

				Convey("Then it should have stopped the service", func() {
					So(services.Servers, ShouldContainKey, agent.Name)
					So(stateCode(services.Servers[agent.Name]), ShouldEqual, utils.StateOffline)
				})
			})

			Convey("Given an incorrect name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/stop", nil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 404 code", func() {
					So(w.Code, ShouldEqual, http.StatusNotFound)
					So(w.Body.String(), ShouldEqual, "server \"toto\" not found\n")
				})
			})
		})
	})
}

func TestRestartServer(t *testing.T) {
	Convey("Given the server stop handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_agent_update_logger")
		db := database.TestDatabase(c)
		handle := restartServer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 agent", func() {
			agent := model.LocalAgent{
				Name: "local server", Protocol: testProto1,
				Address: types.Addr("localhost", 1),
			}
			So(db.Insert(&agent).Run(), ShouldBeNil)

			serv := makeAndStartTestService()
			services.Servers[agent.Name] = serv
			defer delete(services.Servers, agent.Name)

			Convey("Given a valid name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/restart", nil)
				r = mux.SetURLVars(r, map[string]string{"server": agent.Name})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 202 code", func() {
					So(w.Body.String(), ShouldBeEmpty)
					So(w.Code, ShouldEqual, http.StatusAccepted)
				})

				Convey("Then it should have restarted the service", func() {
					So(services.Servers, ShouldContainKey, agent.Name)
					So(stateCode(serv), ShouldEqual, utils.StateRunning)
					So(serv.stopped, ShouldBeTrue)
				})
			})

			Convey("Given an incorrect name parameter", func() {
				r := httptest.NewRequest(http.MethodPatch, "/servers/{server}/restart", nil)
				r = mux.SetURLVars(r, map[string]string{"server": "toto"})

				handle.ServeHTTP(w, r)

				Convey("Then it should have replied with a 404 code", func() {
					So(w.Code, ShouldEqual, http.StatusNotFound)
					So(w.Body.String(), ShouldEqual, "server \"toto\" not found\n")
				})
			})
		})
	})
}
