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

func partnerInfoString(p *rest.OutRemoteAgent) string {
	return "● Partner " + p.Name + "\n" +
		"  -Protocol:         " + p.Protocol + "\n" +
		"  -Configuration:    " + string(p.ProtoConfig) + "\n" +
		"  -Authorized rules\n" +
		"   ├─Sending:   " + strings.Join(p.AuthorizedRules.Sending, ", ") + "\n" +
		"   └─Reception: " + strings.Join(p.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetPartner(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &partnerGet{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			partner := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(partner), ShouldBeNil)

			send := &model.Rule{Name: "send", IsSend: true, Path: "send_path"}
			So(db.Create(send), ShouldBeNil)
			receive := &model.Rule{Name: "receive", IsSend: false, Path: "rcv_path"}
			So(db.Create(receive), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Create(sendAll), ShouldBeNil)

			sAccess := &model.RuleAccess{RuleID: send.ID,
				ObjectType: partner.TableName(), ObjectID: partner.ID}
			So(db.Create(sAccess), ShouldBeNil)
			rAccess := &model.RuleAccess{RuleID: receive.ID,
				ObjectType: partner.TableName(), ObjectID: partner.ID}
			So(db.Create(rAccess), ShouldBeNil)

			Convey("Given a valid partner name", func() {
				args := []string{partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the partner's info", func() {
						rules := &rest.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						p := rest.FromRemoteAgent(partner, rules)
						So(getOutput(), ShouldEqual, partnerInfoString(p))
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})
				})
			})
		})
	})
}

func TestAddPartner(t *testing.T) {

	Convey("Testing the partner 'add' command", t, func() {
		out = testFile()
		command := &partnerAdd{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given valid flags", func() {
				args := []string{"-n", "server_name", "-p", "test",
					"-c", `{"key":"val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner was added", func() {
						So(getOutput(), ShouldEqual, "The partner '"+command.Name+
							"' was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						partner := &model.RemoteAgent{
							Name:        command.Name,
							Protocol:    command.Protocol,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "server_name", "-p", "invalid",
					"-c", `{"key":"val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol")
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "server_name", "-p", "fail",
					"-c", `{"key":"val"}`}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "invalid partner configuration: test fail")
					})
				})
			})
		})
	})
}

func TestListPartners(t *testing.T) {

	Convey("Testing the partner 'list' command", t, func() {
		out = testFile()
		command := &partnerList{}

		Convey("Given a gateway with 2 distant partners", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			partner1 := &model.RemoteAgent{
				Name:        "remote_agent1",
				Protocol:    "test",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(partner1), ShouldBeNil)

			partner2 := &model.RemoteAgent{
				Name:        "remote_agent2",
				Protocol:    "test2",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(partner2), ShouldBeNil)

			p1 := rest.FromRemoteAgent(partner1, &rest.AuthorizedRules{})
			p2 := rest.FromRemoteAgent(partner2, &rest.AuthorizedRules{})

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the partners' info", func() {
						So(getOutput(), ShouldEqual, "Partners:\n"+
							partnerInfoString(p1)+partnerInfoString(p2))
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				args := []string{"-l", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should only display 1 partner's info", func() {
						So(getOutput(), ShouldEqual, "Partners:\n"+
							partnerInfoString(p1))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				args := []string{"-o", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should NOT display the 1st partner's info", func() {
						So(getOutput(), ShouldEqual, "Partners:\n"+
							partnerInfoString(p2))
					})
				})
			})

			Convey("Given a 'sort' parameter of 'name-'", func() {
				args := []string{"-s", "name-"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the partners' info in reverse", func() {
						So(getOutput(), ShouldEqual, "Partners:\n"+
							partnerInfoString(p2)+partnerInfoString(p1))
					})
				})
			})

			Convey("Given the 'protocol' parameter is set to 'test'", func() {
				args := []string{"-p", "test"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display all partners using that protocol", func() {
						So(getOutput(), ShouldEqual, "Partners:\n"+
							partnerInfoString(p1))
					})
				})
			})
		})
	})
}

func TestDeletePartner(t *testing.T) {

	Convey("Testing the partner 'delete' command", t, func() {
		out = testFile()
		command := &partnerDelete{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			partner := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(partner), ShouldBeNil)

			Convey("Given a valid partner name", func() {
				args := []string{partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner was deleted", func() {
						So(getOutput(), ShouldEqual, "The partner '"+partner.Name+
							"' was successfully deleted from the database.\n")
					})

					Convey("Then the partner should have been removed", func() {
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the partner should still exist", func() {
						exists, err := db.Exists(partner)
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
		command := &partnerUpdate{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: []byte(`{"key":"val"}`),
			}
			So(db.Create(partner), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-n", "new_partner", "-p", "test2",
					"-c", `{"updated_key":"updated_val"}`, partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the "+
						"partner was updated", func() {
						So(getOutput(), ShouldEqual, "The partner 'new_partner' "+
							"was successfully updated.\n")
					})

					Convey("Then the old partner should have been removed", func() {
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})

					Convey("Then the new partner should exist", func() {
						newPartner := &model.RemoteAgent{
							ID:          partner.ID,
							Name:        command.Name,
							Protocol:    command.Protocol,
							ProtoConfig: []byte(command.ProtoConfig),
						}
						exists, err := db.Exists(newPartner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "new_partner", "-p", "invalid",
					"-c", `{"updated_key":"updated_val"}`, partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "new_partner", "-p", "fail",
					"-c", `{"updated_key":"updated_val"}`, partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "invalid partner configuration: test fail")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("Given an non-existing name", func() {
				args := []string{"-n", "new_partner", "-p", "test2",
					"-c", `{"updated_key":"updated_val"}`, "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(partner)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}
