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

func TestGetPartner(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &partnerGetCommand{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			partner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&partner)
			So(err, ShouldBeNil)

			Convey("Given a valid partner ID", func() {
				id := fmt.Sprint(partner.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the partner's info", func() {
						var config bytes.Buffer
						err := json.Indent(&config, partner.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote agent n°1:\n"+
							"          Name: "+partner.Name+"\n"+
							"      Protocol: "+partner.Protocol+"\n"+
							" Configuration: "+config.String()+"\n",
						)
					})
				})
			})

			Convey("Given an invalid partner ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.RemoteAgentsPath+
							"/1000' does not exist")

					})
				})
			})
		})
	})
}

func TestAddPartner(t *testing.T) {

	Convey("Testing the partner 'add' command", t, func() {
		out = testFile()
		command := &partnerAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given valid flags", func() {
				command.Name = "remote_agent"
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

					Convey("Then is should display a message saying the partner was added", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The partner '"+command.Name+
							"' was successfully added. It can be consulted at "+
							"the address: "+gw.URL+admin.APIPath+
							rest.RemoteAgentsPath+"/1\n")
					})

					Convey("Then the new partner should have been added", func() {
						partner := model.RemoteAgent{
							Name:        command.Name,
							Protocol:    command.Protocol,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(&partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				command.Name = "partner"
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
							"The agent's protocol must be one of: [sftp r66]")
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				command.Name = "partner"
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

func TestListPartners(t *testing.T) {

	Convey("Testing the partner 'list' command", t, func() {
		out = testFile()
		command := &partnerListCommand{}
		_, err := flags.ParseArgs(command, nil)
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 distant partners", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			partner1 := model.RemoteAgent{
				Name:        "remote_agent1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&partner1)
			So(err, ShouldBeNil)

			partner2 := model.RemoteAgent{
				Name:        "remote_agent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"titi"}`),
			}
			err = db.Create(&partner2)
			So(err, ShouldBeNil)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the partners' info", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, partner1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						var config2 bytes.Buffer
						err = json.Indent(&config2, partner2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote agents:\n"+
							"Remote agent n°1:\n"+
							"          Name: "+partner1.Name+"\n"+
							"      Protocol: "+partner1.Protocol+"\n"+
							" Configuration: "+config1.String()+"\n"+
							"Remote agent n°2:\n"+
							"          Name: "+partner2.Name+"\n"+
							"      Protocol: "+partner2.Protocol+"\n"+
							" Configuration: "+config2.String()+"\n",
						)
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

					Convey("Then it should only display 1 partner's info", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, partner1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote agents:\n"+
							"Remote agent n°1:\n"+
							"          Name: "+partner1.Name+"\n"+
							"      Protocol: "+partner1.Protocol+"\n"+
							" Configuration: "+config1.String()+"\n",
						)
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

					Convey("Then it should NOT display the 1st partner's info", func() {
						var config2 bytes.Buffer
						err := json.Indent(&config2, partner2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote agents:\n"+
							"Remote agent n°2:\n"+
							"          Name: "+partner2.Name+"\n"+
							"      Protocol: "+partner2.Protocol+"\n"+
							" Configuration: "+config2.String()+"\n",
						)
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

					Convey("Then it should display the partners' info in reverse", func() {
						var config1 bytes.Buffer
						err := json.Indent(&config1, partner1.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						var config2 bytes.Buffer
						err = json.Indent(&config2, partner2.ProtoConfig, "  ", "  ")
						So(err, ShouldBeNil)

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote agents:\n"+
							"Remote agent n°2:\n"+
							"          Name: "+partner2.Name+"\n"+
							"      Protocol: "+partner2.Protocol+"\n"+
							" Configuration: "+config2.String()+"\n"+
							"Remote agent n°1:\n"+
							"          Name: "+partner1.Name+"\n"+
							"      Protocol: "+partner1.Protocol+"\n"+
							" Configuration: "+config1.String()+"\n",
						)
					})
				})
			})
		})
	})
}

func TestDeletePartner(t *testing.T) {

	Convey("Testing the partner 'delete' command", t, func() {
		out = testFile()
		command := &partnerDeleteCommand{}

		Convey("Given a gateway with 1 distant partner", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			partner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&partner)
			So(err, ShouldBeNil)

			Convey("Given a valid partner ID", func() {
				id := fmt.Sprint(partner.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the partner was deleted", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The partner n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the partner should have been removed", func() {
						exists, err := db.Exists(&partner)
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
							addr+admin.APIPath+rest.RemoteAgentsPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should still exist", func() {
						exists, err := db.Exists(&partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestUpdatePartner(t *testing.T) {

	Convey("Testing the partner 'delete' command", t, func() {
		out = testFile()
		command := &partnerUpdateCommand{}

		Convey("Given a gateway with 1 distant partner", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			partner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&partner)
			So(err, ShouldBeNil)

			command.Name = "new_remote_agent"
			command.Protocol = "sftp"
			command.ProtoConfig = `{"address":"localhost","port":2023,"root":"titi"}`

			Convey("Given a valid partner ID", func() {
				id := fmt.Sprint(partner.ID)

				Convey("Given all valid flags", func() {

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the partner was updated", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The partner n°"+id+
								" was successfully updated\n")
						})

						Convey("Then the old partner should have been removed", func() {
							exists, err := db.Exists(&partner)
							So(err, ShouldBeNil)
							So(exists, ShouldBeFalse)
						})

						Convey("Then the new partner should exist", func() {
							newPartner := model.RemoteAgent{
								ID:          partner.ID,
								Name:        command.Name,
								Protocol:    command.Protocol,
								ProtoConfig: []byte(command.ProtoConfig),
							}
							exists, err := db.Exists(&newPartner)
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
								"The agent's protocol must be one of: [sftp r66]")
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
								"MarshalJSON for type json.RawMessage: "+
								"unexpected end of JSON input")
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
							addr+admin.APIPath+rest.RemoteAgentsPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(&partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}
