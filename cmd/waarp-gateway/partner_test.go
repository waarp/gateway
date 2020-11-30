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

func partnerInfoString(p *api.OutPartner) string {
	return "● Partner " + p.Name + "\n" +
		"    Protocol:      " + p.Protocol + "\n" +
		"    Address:       " + p.Address + "\n" +
		"    Configuration: " + string(p.ProtoConfig) + "\n" +
		"    Authorized rules\n" +
		"    ├─Sending:   " + strings.Join(p.AuthorizedRules.Sending, ", ") + "\n" +
		"    └─Reception: " + strings.Join(p.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetPartner(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &partnerGet{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner_name",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
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
						rules := &api.AuthorizedRules{
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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{"-n", "server_name", "-p", "test",
					"-c", `{}`, "-a", "localhost:1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner was added", func() {
						So(getOutput(), ShouldEqual, "The partner "+command.Name+
							" was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						var parts []model.RemoteAgent
						So(db.Select(&parts, nil), ShouldBeNil)
						So(len(parts), ShouldEqual, 1)

						exp := model.RemoteAgent{
							ID:          1,
							Name:        "server_name",
							Protocol:    "test",
							ProtoConfig: json.RawMessage(`{}`),
							Address:     "localhost:1",
						}
						So(parts[0], ShouldResemble, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "server_name", "-p", "invalid",
					"-c", `{}`, "-a", "localhost:1"}

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
					"-c", `{"unknown":"val"}`, "-a", "localhost:1"}

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
					"-c", `{"key":"val"}`, "-a", "invalid_address"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid partner address")
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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner1 := &model.RemoteAgent{
				Name:        "partner1",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner1), ShouldBeNil)

			partner2 := &model.RemoteAgent{
				Name:        "partner2",
				Protocol:    "test2",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Create(partner2), ShouldBeNil)

			p1 := rest.FromRemoteAgent(partner1, &api.AuthorizedRules{})
			p2 := rest.FromRemoteAgent(partner2, &api.AuthorizedRules{})

			Convey("Given no parameters", func() {
				var args []string

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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "existing",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)

			Convey("Given a valid partner name", func() {
				args := []string{partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner was deleted", func() {
						So(getOutput(), ShouldEqual, "The partner "+partner.Name+
							" was successfully deleted.\n")
					})

					Convey("Then the partner should have been removed", func() {
						var parts []model.RemoteAgent
						So(db.Select(&parts, nil), ShouldBeNil)
						So(parts, ShouldBeEmpty)
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
						var parts []model.RemoteAgent
						So(db.Select(&parts, nil), ShouldBeNil)
						So(len(parts), ShouldEqual, 1)

						So(parts[0], ShouldResemble, *partner)
					})
				})
			})
		})
	})
}

func TestUpdatePartner(t *testing.T) {

	Convey("Testing the partner 'update' command", t, func() {
		out = testFile()
		command := &partnerUpdate{}

		Convey("Given a gateway with 1 distant partner", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-n", "new_partner", "-p", "test2",
					"-a", "localhost:1", partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the "+
						"partner was updated", func() {
						So(getOutput(), ShouldEqual, "The partner new_partner "+
							"was successfully updated.\n")
					})

					Convey("Then the partner should have been updated", func() {
						var parts []model.RemoteAgent
						So(db.Select(&parts, nil), ShouldBeNil)
						So(len(parts), ShouldEqual, 1)

						exp := model.RemoteAgent{
							ID:          partner.ID,
							Name:        "new_partner",
							Protocol:    "test2",
							Address:     "localhost:1",
							ProtoConfig: json.RawMessage(`{}`),
						}
						So(parts[0], ShouldResemble, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{"-n", "new_partner", "-p", "invalid",
					"-a", "localhost:1", partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "unknown protocol 'invalid'")
					})

					Convey("Then the partner should stay unchanged", func() {
						So(db.Get(partner), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{"-n", "new_partner", "-p", "fail",
					"-c", `unknown:val`, "-a", "localhost:1", partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, `failed to parse protocol `+
							`configuration: json: unknown field "unknown"`)
					})

					Convey("Then the partner should stay unchanged", func() {
						So(db.Get(partner), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{"-n", "new_partner", "-p", "fail",
					"-a", "invalid_address", partner.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid "+
							"partner address")
					})

					Convey("Then the partner should stay unchanged", func() {
						So(db.Get(partner), ShouldBeNil)
					})
				})
			})

			Convey("Given an non-existing name", func() {
				args := []string{"-n", "new_partner", "-p", "test2",
					"-a", "localhost:1", "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the partner should stay unchanged", func() {
						So(db.Get(partner), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestAuthorizePartner(t *testing.T) {

	Convey("Testing the partner 'authorize' command", t, func() {
		out = testFile()
		command := &partnerAuthorize{}

		Convey("Given a gateway with 1 distant partner and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid partner & rule names", func() {
				args := []string{partner.Name, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner can use the rule", func() {
						So(getOutput(), ShouldEqual, "Usage of the "+direction(rule)+" rule '"+
							rule.Name+"' is now restricted.\nThe partner "+partner.Name+
							" is now allowed to use the "+direction(rule)+" rule "+rule.Name+
							" for transfers.\n")
					})

					Convey("Then the permission should have been added", func() {
						access := &model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   partner.ID,
							ObjectType: partner.TableName(),
						}
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{partner.Name, "toto", direction(rule)}

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

			Convey("Given an invalid partner name", func() {
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var a []model.RuleAccess
						So(db.Select(a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})
		})
	})
}

func TestRevokePartner(t *testing.T) {

	Convey("Testing the partner 'revoke' command", t, func() {
		out = testFile()
		command := &partnerRevoke{}

		Convey("Given a gateway with 1 distant partner and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   partner.ID,
				ObjectType: partner.TableName(),
			}
			So(db.Create(access), ShouldBeNil)

			Convey("Given a valid partner & rule names", func() {
				args := []string{partner.Name, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner cannot use the rule", func() {
						So(getOutput(), ShouldEqual, "The partner "+partner.Name+
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
				args := []string{partner.Name, "toto", direction(rule)}

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

			Convey("Given an invalid partner name", func() {
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})
		})
	})
}
