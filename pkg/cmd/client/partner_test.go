package wg

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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
		command := &PartnerGet{}

		Convey("Given a gateway with 1 distant partner", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner_name",
				Protocol: "sftp",
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			send := &model.Rule{Name: "send_rule", IsSend: true, Path: "send_path"}
			So(db.Insert(send).Run(), ShouldBeNil)
			receive := &model.Rule{Name: "receive", IsSend: false, Path: "rcv_path"}
			So(db.Insert(receive).Run(), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Insert(sendAll).Run(), ShouldBeNil)

			sAccess := &model.RuleAccess{
				RuleID:     send.ID,
				ObjectType: partner.TableName(), ObjectID: partner.ID,
			}
			So(db.Insert(sAccess).Run(), ShouldBeNil)
			rAccess := &model.RuleAccess{
				RuleID:     receive.ID,
				ObjectType: partner.TableName(), ObjectID: partner.ID,
			}
			So(db.Insert(rAccess).Run(), ShouldBeNil)

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
		command := &PartnerAdd{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				args := []string{
					"--name", "server_name", "--protocol", testProto1,
					"--address", "localhost:1", "--config", "key1:val1",
					"--config", "key2:val2",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner was added", func() {
						So(getOutput(), ShouldEqual, "The partner "+command.Name+
							" was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)

						exp := model.RemoteAgent{
							ID:          1,
							Name:        "server_name",
							Protocol:    testProto1,
							ProtoConfig: json.RawMessage(`{"key1":"val1","key2":"val2"}`),
							Address:     "localhost:1",
						}
						So(partners, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{
					"--name", "server_name", "--protocol", "invalid",
					"--address", "localhost:1",
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
					"--config", `key:val`, "--address", "localhost:1",
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
					"--config", `key:val`, "--address", "invalid_address",
				}

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
		command := &PartnerList{}

		Convey("Given a gateway with 2 distant partners", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner1 := &model.RemoteAgent{
				Name:     "partner1",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(partner1).Run(), ShouldBeNil)

			partner2 := &model.RemoteAgent{
				Name:     "partner2",
				Protocol: testProto2,
				Address:  "localhost:2",
			}
			So(db.Insert(partner2).Run(), ShouldBeNil)

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
				args := []string{"--protocol", testProto1}

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
		command := &PartnerDelete{}

		Convey("Given a gateway with 1 distant partner", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "existing",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

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
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldBeEmpty)
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
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldContain, *partner)
					})
				})
			})
		})
	})
}

func TestUpdatePartner(t *testing.T) {
	Convey("Testing the partner 'update' command", t, func() {
		out = testFile()
		command := &PartnerUpdate{}

		Convey("Given a gateway with 1 distant partner", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{
					partner.Name,
					"--name", "new_partner", "--protocol", testProto2,
					"--address", "localhost:1",
				}

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
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)

						exp := model.RemoteAgent{
							ID:          partner.ID,
							Name:        "new_partner",
							Protocol:    testProto2,
							Address:     "localhost:1",
							ProtoConfig: json.RawMessage(`{}`),
						}
						So(partners, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid protocol", func() {
				args := []string{
					partner.Name,
					"--name", "new_partner", "--protocol", "invalid",
					"--address", "localhost:1",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, "unknown protocol 'invalid'")
					})

					Convey("Then the partner should stay unchanged", func() {
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldContain, *partner)
					})
				})
			})

			Convey("Given an invalid configuration", func() {
				args := []string{
					partner.Name,
					"--name", "new_partner", "--protocol", testProtoErr,
					"--config", `key:val`, "--address", "localhost:1",
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

					Convey("Then the partner should stay unchanged", func() {
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldContain, *partner)
					})
				})
			})

			Convey("Given an invalid address", func() {
				args := []string{
					partner.Name,
					"--name", "new_partner", "--protocol", testProtoErr,
					"--address", "invalid_address",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'invalid_address' is not a valid "+
							"partner address")
					})

					Convey("Then the partner should stay unchanged", func() {
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldContain, *partner)
					})
				})
			})

			Convey("Given an non-existing name", func() {
				args := []string{
					"toto",
					"--name", "new_partner", "--protocol", testProto2,
					"--address", "localhost:1",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the partner should stay unchanged", func() {
						var partners model.RemoteAgents
						So(db.Select(&partners).Run(), ShouldBeNil)
						So(partners, ShouldContain, *partner)
					})
				})
			})
		})
	})
}

func TestAuthorizePartner(t *testing.T) {
	Convey("Testing the partner 'authorize' command", t, func() {
		out = testFile()
		command := &PartnerAuthorize{}

		Convey("Given a gateway with 1 distant partner and 1 rule", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)

						exp := model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   partner.ID,
							ObjectType: partner.TableName(),
						}
						So(accesses, ShouldContain, exp)
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
						So(err, ShouldBeError, "send rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
					})
				})
			})
		})
	})
}

func TestRevokePartner(t *testing.T) {
	Convey("Testing the partner 'revoke' command", t, func() {
		out = testFile()
		command := &PartnerRevoke{}

		Convey("Given a gateway with 1 distant partner and 1 rule", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   partner.ID,
				ObjectType: partner.TableName(),
			}
			So(db.Insert(access).Run(), ShouldBeNil)

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
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldBeEmpty)
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
						So(err, ShouldBeError, "send rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldContain, *access)
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

					Convey("Then the permission should NOT have been removed", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldContain, *access)
					})
				})
			})
		})
	})
}
