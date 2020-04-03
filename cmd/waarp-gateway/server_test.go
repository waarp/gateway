package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func agentInfoString(s *rest.OutAgent) string {
	var config bytes.Buffer
	_ = json.Indent(&config, s.ProtoConfig, "    ", "  ")
	return "● " + s.Name + " (ID " + fmt.Sprint(s.ID) + ")\n" +
		"  -Protocol     : " + s.Protocol + "\n" +
		"  -Configuration: " + config.String() + "\n"
}

func TestGetServer(t *testing.T) {

	Convey("Testing the server 'get' command", t, func() {
		out = testFile()
		command := &serverGetCommand{}

		Convey("Given a gateway with 1 local server", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			server := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given a valid server ID", func() {
				id := fmt.Sprint(server.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the server's info", func() {
						var config bytes.Buffer
						err := json.Indent(&config, server.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)

						s := rest.FromLocalAgent(server)
						So(string(cont), ShouldEqual, agentInfoString(s))
					})
				})
			})

			Convey("Given an invalid server ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.LocalAgentsPath+
							"/1000' does not exist")

					})
				})
			})
		})
	})
}

func TestAddServer(t *testing.T) {

	Convey("Testing the server 'add' command", t, func() {
		out = testFile()
		command := &serverAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given valid flags", func() {
				command.Name = "local_agent"
				command.Protocol = "sftp"
				command.ProtoConfig = `{"address":"localhost","port":2022,"root":"toto"}`

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the server was added", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The server '"+command.Name+
							"' was successfully added. It can be consulted at "+
							"the address: "+gw.URL+admin.APIPath+
							rest.LocalAgentsPath+"/1\n")
					})

					Convey("Then the new server should have been added", func() {
						server := &model.LocalAgent{
							Name:        command.Name,
							Protocol:    command.Protocol,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				command.Name = "server"
				command.Protocol = "not a protocol"
				command.ProtoConfig = "{}"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "400 - Invalid request: "+
							"Invalid agent configuration: unknown protocol")
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				command.Name = "server"
				command.Protocol = "sftp"
				command.ProtoConfig = "{"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "json: error calling "+
							"MarshalJSON for type json.RawMessage: unexpected "+
							"end of JSON input")
					})
				})
			})
		})
	})
}

func TestListServers(t *testing.T) {

	Convey("Testing the server 'list' command", t, func() {
		out = testFile()
		command := &serverListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 local servers", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			server1 := &model.LocalAgent{
				Name:        "local_agent1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(server1), ShouldBeNil)

			server2 := &model.LocalAgent{
				Name:        "local_agent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"titi"}`),
			}
			So(db.Create(server2), ShouldBeNil)

			s1 := rest.FromLocalAgent(server1)
			s2 := rest.FromLocalAgent(server2)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the servers' info", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, server1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						var config2 bytes.Buffer
						err = json.Indent(&config2, server2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Local agents:\n"+
							agentInfoString(s1)+agentInfoString(s2))
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				command.Limit = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should only display 1 server's info", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, server1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Local agents:\n"+
							agentInfoString(s1))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				command.Offset = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should NOT display the 1st server's info", func() {
						var config2 bytes.Buffer
						err := json.Indent(&config2, server2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Local agents:\n"+
							agentInfoString(s2))
					})
				})
			})

			Convey("Given that the 'desc' flag is set", func() {
				command.DescOrder = true

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the servers' info in reverse", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, server1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						var config2 bytes.Buffer
						err = json.Indent(&config2, server2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Local agents:\n"+
							agentInfoString(s2)+agentInfoString(s1))
					})
				})
			})
		})
	})
}

func TestDeleteServer(t *testing.T) {

	Convey("Testing the server 'delete' command", t, func() {
		out = testFile()
		command := &serverDeleteCommand{}

		Convey("Given a gateway with 1 local server", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			server := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(server), ShouldBeNil)

			Convey("Given a valid server ID", func() {
				id := fmt.Sprint(server.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the server was deleted", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The server n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the server should have been removed", func() {
						exists, err := db.Exists(server)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.LocalAgentsPath+
							"/1000' does not exist")
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
		command := &serverUpdateCommand{}

		Convey("Given a gateway with 1 local server", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			server := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(server), ShouldBeNil)

			command.Name = "new_local_agent"
			command.Protocol = "sftp"
			command.ProtoConfig = `{"address":"localhost","port":2023,"root":"titi"}`

			Convey("Given a valid server ID", func() {
				id := fmt.Sprint(server.ID)

				Convey("Given all valid flags", func() {

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the server was updated", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The server n°"+id+
								" was successfully updated\n")
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
					command.Protocol = "not a protocol"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"Invalid agent configuration: unknown protocol")
						})
					})
				})

				Convey("Given an invalid configuration", func() {
					command.ProtoConfig = "{"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "json: error calling "+
								"MarshalJSON for type json.RawMessage: unexpected "+
								"end of JSON input")
						})
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.LocalAgentsPath+
							"/1000' does not exist")
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
