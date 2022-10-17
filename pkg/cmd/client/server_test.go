package wg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func serverInfoString(s *api.OutServer) string {
	status := "Enabled"
	if !s.Enabled {
		status = "Disabled"
	}

	return `● Server "` + s.Name + `" [` + status + "]\n" +
		"    Protocol:               " + s.Protocol + "\n" +
		"    Address:                " + s.Address + "\n" +
		"    Root directory:         " + s.RootDir + "\n" +
		"    Receive directory:      " + s.ReceiveDir + "\n" +
		"    Send directory:         " + s.SendDir + "\n" +
		"    Temp receive directory: " + s.TmpReceiveDir + "\n" +
		"    Configuration:          " + string(s.ProtoConfig) + "\n" +
		"    Authorized rules\n" +
		"    ├─Sending:              " + strings.Join(s.AuthorizedRules.Sending, ", ") + "\n" +
		"    └─Reception:            " + strings.Join(s.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetServer(t *testing.T) {
	Convey("Testing the server 'get' command", t, func() {
		out = testFile()
		command := &ServerGet{}

		Convey("Given a gateway with 1 local server", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:          "server_name",
				Protocol:      testProto1,
				RootDir:       "/server/root",
				ReceiveDir:    "/in",
				SendDir:       "/out",
				TmpReceiveDir: "/tmp",
				Address:       "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			send := &model.Rule{Name: "send", IsSend: true, Path: "send_path"}
			So(db.Insert(send).Run(), ShouldBeNil)
			receive := &model.Rule{Name: "receive", IsSend: false, Path: "rcv_path"}
			So(db.Insert(receive).Run(), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Insert(sendAll).Run(), ShouldBeNil)

			sAccess := &model.RuleAccess{
				RuleID:     send.ID,
				ObjectType: server.TableName(), ObjectID: server.ID,
			}
			So(db.Insert(sAccess).Run(), ShouldBeNil)
			rAccess := &model.RuleAccess{
				RuleID:     receive.ID,
				ObjectType: server.TableName(), ObjectID: server.ID,
			}
			So(db.Insert(rAccess).Run(), ShouldBeNil)

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
		command := &ServerAdd{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{
					"--name", "server_name", "--protocol", testProto1,
					"--root-dir", "root", "--receive-dir", "rcv_dir",
					"--send-dir", "snd_dir", "--tmp-dir", "tmp_dir",
					"--address", "localhost:1",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The server server_name "+
							"was successfully added.\n")
					})

					Convey("Then the new server should have been added", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)

						exp := model.LocalAgent{
							ID:            1,
							Owner:         conf.GlobalConfig.GatewayName,
							Name:          command.Name,
							Address:       command.Address,
							Protocol:      command.Protocol,
							RootDir:       *command.RootDir,
							ReceiveDir:    *command.ReceiveDir,
							SendDir:       *command.SendDir,
							TmpReceiveDir: *command.TempRcvDir,
							ProtoConfig:   json.RawMessage(`{}`),
						}
						So(servers, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{
					"--name", "server_name", "--protocol", "invalid",
					"--root-dir", "server/root", "--address", "localhost:1",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, "unknown protocol 'invalid'")
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{
					"--name", "server_name", "--protocol", testProtoErr,
					"--root-dir", "server/root", "--config", "key:0",
					"--address", "localhost:1",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, `failed to parse protocol `+
							`configuration: json: unknown field "key"`)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{
					"--name", "server_name", "--protocol", testProtoErr,
					"--root-dir", "server/root", "--address", "invalid_address",
				}

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

			Convey("Given a new R66 server", func() {
				args := []string{
					"--name", "r66_server", "--protocol", config.ProtocolR66,
					"--root-dir", "root_dir", "--receive-dir", "rcv_dir",
					"--send-dir", "snd_dir", "--tmp-dir", "tmp_dir",
					"--address", "localhost:1", "--config", "blockSize:256",
					"--config", "serverPassword:sesame",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The server r66_server "+
							"was successfully added.\n")
					})

					Convey("Then the new server should have been added", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(len(servers), ShouldEqual, 1)

						var r66Conf config.R66ProtoConfig
						So(json.Unmarshal(servers[0].ProtoConfig, &r66Conf), ShouldBeNil)
						pwd, err := utils.AESDecrypt(database.GCM, r66Conf.ServerPassword)
						So(err, ShouldBeNil)

						So(pwd, ShouldEqual, "sesame")
						r66Conf.ServerPassword = "sesame"
						servers[0].ProtoConfig, err = json.Marshal(r66Conf)
						So(err, ShouldBeNil)

						exp := model.LocalAgent{
							ID:            1,
							Owner:         conf.GlobalConfig.GatewayName,
							Name:          "r66_server",
							Address:       "localhost:1",
							Protocol:      config.ProtocolR66,
							RootDir:       "root_dir",
							ReceiveDir:    "rcv_dir",
							SendDir:       "snd_dir",
							TmpReceiveDir: "tmp_dir",
							ProtoConfig:   json.RawMessage(`{"blockSize":256,"serverPassword":"sesame"}`),
						}
						So(servers[0], ShouldResemble, exp)
					})
				})
			})
		})
	})
}

func TestListServers(t *testing.T) {
	Convey("Testing the server 'list' command", t, func() {
		out = testFile()
		command := &ServerList{}

		Convey("Given a gateway with 2 local servers", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server1 := &model.LocalAgent{
				Name:          "server1",
				Protocol:      testProto1,
				RootDir:       "/test/root1",
				ReceiveDir:    "/test/in1",
				SendDir:       "/test/out1",
				TmpReceiveDir: "/test/tmp1",
				ProtoConfig:   json.RawMessage(`{}`),
				Address:       "localhost:1",
			}
			So(db.Insert(server1).Run(), ShouldBeNil)

			server2 := &model.LocalAgent{
				Name:          "server2",
				Protocol:      testProto2,
				RootDir:       "/test/root2",
				ReceiveDir:    "/test/in2",
				SendDir:       "/test/out2",
				TmpReceiveDir: "/test/tmp2",
				ProtoConfig:   json.RawMessage(`{}`),
				Address:       "localhost:2",
			}
			So(db.Insert(server2).Run(), ShouldBeNil)

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
				args := []string{"-p", testProto1}

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
		command := &ServerDelete{}

		Convey("Given a gateway with 1 local server", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server_name",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

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
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldBeEmpty)
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
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldContain, *server)
					})
				})
			})
		})
	})
}

func TestUpdateServer(t *testing.T) {
	Convey("Testing the server 'delete' command", t, func() {
		out = testFile()
		command := &ServerUpdate{}

		Convey("Given a gateway with 1 local server", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{
					server.Name,
					"--name", "new_server", "--protocol", testProto2,
					"--address", "localhost:2",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was updated", func() {
						So(getOutput(), ShouldEqual, "The server new_server "+
							"was successfully updated.\n")
					})

					Convey("Then the server should have been updated", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)

						exp := model.LocalAgent{
							ID:          server.ID,
							Owner:       server.Owner,
							Name:        *command.Name,
							Address:     *command.Address,
							Protocol:    *command.Protocol,
							ProtoConfig: json.RawMessage(`{}`),
						}
						So(servers, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{
					server.Name,
					"--name", "new_server", "--protocol", "invalid",
					"--address", "localhost:2",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, "unknown protocol 'invalid'")
					})

					Convey("Then the server should stay unchanged", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldContain, *server)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{
					server.Name,
					"--name", "new_server", "--protocol", testProtoErr,
					"--config", "key:val", "--address", "localhost:2",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, "failed to parse protocol "+
							`configuration: json: unknown field "key"`)
					})

					Convey("Then the server should stay unchanged", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldContain, *server)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{
					server.Name,
					"--name", "new_server", "--protocol", testProtoErr,
					"--address", "invalid_address",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid "+
							"server address")
					})

					Convey("Then the server should stay unchanged", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldContain, *server)
					})
				})
			})

			Convey("Given a non-existing name", func() {
				args := []string{
					"toto",
					"--name", "new_server", "--protocol", testProto2,
					"--config", "updated_key:updated_val",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the server should stay unchanged", func() {
						var servers model.LocalAgents
						So(db.Select(&servers).Run(), ShouldBeNil)
						So(servers, ShouldContain, *server)
					})
				})
			})
		})
	})
}

func TestAuthorizeServer(t *testing.T) {
	Convey("Testing the server 'authorize' command", t, func() {
		out = testFile()
		command := &ServerAuthorize{}

		Convey("Given a gateway with 1 local server and 1 rule", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)

						access := model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   server.ID,
							ObjectType: server.TableName(),
						}
						So(accesses, ShouldContain, access)
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
						So(err, ShouldBeError, "send rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
					})
				})
			})
		})
	})
}

func TestRevokeServer(t *testing.T) {
	Convey("Testing the server 'revoke' command", t, func() {
		out = testFile()
		command := &ServerRevoke{}

		Convey("Given a gateway with 1 distant server and 1 rule", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   server.ID,
				ObjectType: server.TableName(),
			}
			So(db.Insert(access).Run(), ShouldBeNil)

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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
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
						So(err, ShouldBeError, "send rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldContain, *access)
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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldContain, *access)
					})
				})
			})
		})
	})
}

func TestEnableDisableServer(t *testing.T) {
	const (
		locAgentName = "test_server"
		enablePath   = "/api/servers/" + locAgentName + "/enable"
		disablePath  = "/api/servers/" + locAgentName + "/disable"
	)

	Convey(`Given the server "enable" command`, t, func() {
		out = testFile()
		command := &ServerEnable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   enablePath,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(command, locAgentName), ShouldBeNil)

				Convey("Then it should display a message saying the server was enabled", func() {
					So(getOutput(), ShouldEqual, "The server "+locAgentName+
						" was successfully enabled.\n")
				})
			})
		})
	})

	Convey(`Given the server "disable" command`, t, func() {
		out = testFile()
		command := &ServerDisable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   disablePath,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(command, locAgentName), ShouldBeNil)

				Convey("Then it should display a message saying the server was disabled", func() {
					So(getOutput(), ShouldEqual, "The server "+locAgentName+
						" was successfully disabled.\n")
				})
			})
		})
	})
}
