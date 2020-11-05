package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func serverInfoString(s *api.OutServer) string {
	return "● Server " + s.Name + "\n" +
		"    Protocol:       " + s.Protocol + "\n" +
		"    Address:        " + s.Address + "\n" +
		"    Root:           " + s.Root + "\n" +
		"    In directory:   " + s.InDir + "\n" +
		"    Out directory:  " + s.OutDir + "\n" +
		"    Work directory: " + s.WorkDir + "\n" +
		"    Configuration:  " + string(s.ProtoConfig) + "\n" +
		"    Authorized rules\n" +
		"    ├─Sending:   " + strings.Join(s.AuthorizedRules.Sending, ", ") + "\n" +
		"    └─Reception: " + strings.Join(s.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetServer(t *testing.T) {

	Convey("Testing the server 'get' command", t, func() {
		out = testFile()
		command := &serverGet{}

		Convey("Given a gateway with 1 local server", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				Root:        "/server/root",
				InDir:       "/server/in",
				OutDir:      "/server/out",
				WorkDir:     "/server/work",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server), ShouldBeNil)

			send := &model.Rule{Name: "send", IsSend: true, Path: "send_path"}
			So(db.Create(send), ShouldBeNil)
			receive := &model.Rule{Name: "receive", IsSend: false, Path: "rcv_path"}
			So(db.Create(receive), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Create(sendAll), ShouldBeNil)

			sAccess := &model.RuleAccess{RuleID: send.ID,
				ObjectType: server.TableName(), ObjectID: server.ID}
			So(db.Create(sAccess), ShouldBeNil)
			rAccess := &model.RuleAccess{RuleID: receive.ID,
				ObjectType: server.TableName(), ObjectID: server.ID}
			So(db.Create(rAccess), ShouldBeNil)

			Convey("Given a valid server name", func() {
				args := []string{server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the server's info", func() {
						rules := &api.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						s := rest.FromLocalAgent(server, rules)
						So(getOutput(), ShouldEqual, serverInfoString(s))
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})
				})
			})
		})
	})
}

func TestAddServer(t *testing.T) {

	Convey("Testing the server 'add' command", t, func() {
		out = testFile()
		command := &serverAdd{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"-n", "server_name", "-p", "test",
					"--root=root", "--in=in_dir", "--out=out_dir",
					"--work=work_dir", "-c", `{}`, "-a", "localhost:1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The server server_name "+
							"was successfully added.\n")
					})

					Convey("Then the new server should have been added", func() {
						exp := model.LocalAgent{
							ID:          1,
							Owner:       database.Owner,
							Name:        command.Name,
							Address:     command.Address,
							Protocol:    command.Protocol,
							Root:        *command.Root,
							InDir:       *command.InDir,
							OutDir:      *command.OutDir,
							WorkDir:     *command.WorkDir,
							ProtoConfig: json.RawMessage(command.ProtoConfig),
						}
						var res []model.LocalAgent
						So(db.Select(&res, nil), ShouldBeNil)
						So(len(res), ShouldEqual, 1)
						So(res[0], ShouldResemble, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "server_name", "-p", "invalid",
					"--root=/server/root", "-c", `{}`, "-a", "localhost:1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol 'invalid'")
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "server_name", "-p", "fail",
					"--root=/server/root", "-c", `{"unknown":"val"}`,
					"-a", "localhost:1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, `failed to parse protocol `+
							`configuration: json: unknown field "unknown"`)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{"-n", "server_name", "-p", "fail",
					"--root=/server/root", "-c", `{}`, "-a", "invalid_address"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid "+
							"server address")
					})
				})
			})
		})
	})
}

func TestListServers(t *testing.T) {

	Convey("Testing the server 'list' command", t, func() {
		out = testFile()
		command := &serverList{}

		Convey("Given a gateway with 2 local servers", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server1 := &model.LocalAgent{
				Name:        "local_agent1",
				Protocol:    "test",
				Root:        "/test/root1",
				InDir:       "/test/in1",
				OutDir:      "/test/out1",
				WorkDir:     "/test/work1",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server1), ShouldBeNil)

			server2 := &model.LocalAgent{
				Name:        "local_agent2",
				Protocol:    "test2",
				Root:        "/test/root2",
				InDir:       "/test/in2",
				OutDir:      "/test/out2",
				WorkDir:     "/test/work2",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Create(server2), ShouldBeNil)

			s1 := rest.FromLocalAgent(server1, &api.AuthorizedRules{})
			s2 := rest.FromLocalAgent(server2, &api.AuthorizedRules{})

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the servers' info", func() {
						So(getOutput(), ShouldEqual, "Servers:\n"+
							serverInfoString(s1)+serverInfoString(s2))
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				args := []string{"-l", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should only display 1 server's info", func() {
						So(getOutput(), ShouldEqual, "Servers:\n"+
							serverInfoString(s1))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				args := []string{"-o", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display all but the 1st server's info", func() {
						So(getOutput(), ShouldEqual, "Servers:\n"+
							serverInfoString(s2))
					})
				})
			})

			Convey("Given a 'sort' parameter of 'name-'", func() {
				args := []string{"-s", "name-"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the servers' info in reverse", func() {
						So(getOutput(), ShouldEqual, "Servers:\n"+
							serverInfoString(s2)+serverInfoString(s1))
					})
				})
			})

			Convey("Given the 'protocol' parameter is set to 'test'", func() {
				args := []string{"-p", "test"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display all servers using that protocol", func() {
						So(getOutput(), ShouldEqual, "Servers:\n"+
							serverInfoString(s1))
					})
				})
			})
		})
	})
}

func TestDeleteServer(t *testing.T) {

	Convey("Testing the server 'delete' command", t, func() {
		out = testFile()
		command := &serverDelete{}

		Convey("Given a gateway with 1 local server", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server_name",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given a valid server name", func() {
				args := []string{server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was deleted", func() {
						So(getOutput(), ShouldEqual, "The server "+server.Name+
							" was successfully deleted.\n")
					})

					Convey("Then the server should have been removed", func() {
						var serv []model.LocalAgent
						So(db.Select(&serv, nil), ShouldBeNil)
						So(serv, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the server should still exist", func() {
						So(db.Get(server), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestUpdateServer(t *testing.T) {

	Convey("Testing the server 'delete' command", t, func() {
		out = testFile()
		command := &serverUpdate{}

		Convey("Given a gateway with 1 local server", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-n", "new_server", "-p", "test2",
					"-c", `{}`, "-a", "localhost:2", server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was updated", func() {
						So(getOutput(), ShouldEqual, "The server new_server "+
							"was successfully updated.\n")
					})

					Convey("Then the server should have been updated", func() {
						var serv []model.LocalAgent
						So(db.Select(&serv, nil), ShouldBeNil)
						So(len(serv), ShouldEqual, 1)

						exp := model.LocalAgent{
							ID:          server.ID,
							Owner:       server.Owner,
							Name:        *command.Name,
							Address:     *command.Address,
							Protocol:    *command.Protocol,
							ProtoConfig: json.RawMessage(*command.ProtoConfig),
						}
						So(serv[0], ShouldResemble, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "new_server", "-p", "invalid",
					"-c", `{}`, "-a", "localhost:2", server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol 'invalid'")
					})

					Convey("Then the server should stay unchanged", func() {
						So(db.Get(server), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "new_server", "-p", "fail",
					"-c", `{"unknown":"val"}`, "-a", "localhost:2", server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "failed to parse protocol "+
							`configuration: json: unknown field "unknown"`)
					})

					Convey("Then the server should stay unchanged", func() {
						So(db.Get(server), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{"-n", "new_server", "-p", "fail",
					"-c", `{}`, "-a", "invalid_address", server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid "+
							"server address")
					})

					Convey("Then the server should stay unchanged", func() {
						So(db.Get(server), ShouldBeNil)
					})
				})
			})

			Convey("Given a non-existing name", func() {
				args := []string{"-n", "new_server", "-p", "test2",
					"-c", `{"updated_key":"updated_val"}`, "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the server should stay unchanged", func() {
						So(db.Get(server), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestAuthorizeServer(t *testing.T) {

	Convey("Testing the server 'authorize' command", t, func() {
		out = testFile()
		command := &serverAuthorize{}

		Convey("Given a gateway with 1 local server and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid server & rule names", func() {
				args := []string{server.Name, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server can use the rule", func() {
						So(getOutput(), ShouldEqual, "Usage of the "+direction(rule)+
							" rule '"+rule.Name+"' is now restricted.\nThe server "+
							server.Name+" is now allowed to use the "+direction(rule)+
							" rule "+rule.Name+" for transfers.\n")
					})

					Convey("Then the permission should have been added", func() {
						access := &model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   server.ID,
							ObjectType: server.TableName(),
						}
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{server.Name, "toto", direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var a []model.RuleAccess
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var a []model.RuleAccess
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})
		})
	})
}

func TestRevokeServer(t *testing.T) {

	Convey("Testing the server 'revoke' command", t, func() {
		out = testFile()
		command := &serverRevoke{}

		Convey("Given a gateway with 1 distant server and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(server), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   server.ID,
				ObjectType: server.TableName(),
			}
			So(db.Create(access), ShouldBeNil)

			Convey("Given a valid server & rule names", func() {
				args := []string{server.Name, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server cannot use the rule", func() {
						So(getOutput(), ShouldEqual, "The server "+server.Name+
							" is no longer allowed to use the "+direction(rule)+" rule "+
							rule.Name+" for transfers.\nUsage of the "+direction(rule)+
							" rule '"+rule.Name+"' is now unrestricted.\n")
					})

					Convey("Then the permission should have been removed", func() {
						var a []model.RuleAccess
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{server.Name, "toto", direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})
		})
	})
}
