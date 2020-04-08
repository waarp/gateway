package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func serverInfoString(s *rest.OutLocalAgent) string {
	return "● Server " + s.Name + "\n" +
		"  -Protocol:         " + s.Protocol + "\n" +
		"  -Root:             " + s.Root + "\n" +
		"  -Configuration:    " + string(s.ProtoConfig) + "\n" +
		"  -Authorized rules\n" +
		"   ├─Sending:   " + strings.Join(s.AuthorizedRules.Sending, ", ") + "\n" +
		"   └─Reception: " + strings.Join(s.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetServer(t *testing.T) {

	Convey("Testing the server 'get' command", t, func() {
		out = testFile()
		command := &serverGet{}

		Convey("Given a gateway with 1 local server", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				Root:        "/server/root",
				ProtoConfig: []byte(`{"key":"val"}`),
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
						rules := &rest.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						s := rest.FromLocalAgent(server, rules)
						So(getOutput(), ShouldEqual, serverInfoString(s))
					})
				})
			})

			Convey("Given an invalid server ID", func() {
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given valid flags", func() {
				args := []string{"-n", "server_name", "-p", "test",
					"r", "/server/root", "-c", `{"key":"val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The server 'server_name' "+
							"was successfully added.\n")
					})

					Convey("Then the new server should have been added", func() {
						server := &model.LocalAgent{
							Name:        command.Name,
							Protocol:    command.Protocol,
							Root:        command.Root,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "server_name", "-p", "invalid",
					"r", "/server/root", "-c", `{"key":"val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)

					Convey("Then it should return an error", func() {
						So(command.Execute(params), ShouldNotBeNil)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "server_name", "-p", "test",
					"r", "/server/root", "-c", `{"key":val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)

					Convey("Then it should return an error", func() {
						So(command.Execute(params), ShouldNotBeNil)
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server1 := &model.LocalAgent{
				Name:        "local_agent1",
				Protocol:    "test",
				Root:        "/test/root1",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(server1), ShouldBeNil)

			server2 := &model.LocalAgent{
				Name:        "local_agent2",
				Protocol:    "test2",
				Root:        "/test/root2",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(server2), ShouldBeNil)

			s1 := rest.FromLocalAgent(server1, &rest.AuthorizedRules{})
			s2 := rest.FromLocalAgent(server2, &rest.AuthorizedRules{})

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

			Convey("Given that the 'desc' flag is set", func() {
				args := []string{"-d"}

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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "server_name",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given a valid server name", func() {
				args := []string{server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was deleted", func() {
						So(getOutput(), ShouldEqual, "The server '"+server.Name+
							"' was successfully deleted from the database.\n")
					})

					Convey("Then the server should have been removed", func() {
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
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
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-n", "new_server", "-p", "test2",
					"-c", `{"updated_key":"updated_val"}`, server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was updated", func() {
						So(getOutput(), ShouldEqual, "The server 'new_server' "+
							"was successfully updated.\n")
					})

					Convey("Then the old server should have been removed", func() {
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})

					Convey("Then the new server should exist", func() {
						newServer := model.LocalAgent{
							ID:          server.ID,
							Owner:       server.Owner,
							Name:        command.Name,
							Protocol:    command.Protocol,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(&newServer)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "new_server", "-p", "invalid",
					"-c", `{"updated_key":"updated_val"}`, server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol")
					})

					Convey("Then the server should stay unchanged", func() {
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "new_server", "-p", "fail",
					"-c", `{"updated_key":"updated_val"}`, server.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "invalid server configuration: test fail")
					})

					Convey("Then the server should stay unchanged", func() {
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
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
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}
