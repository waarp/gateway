package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetRemoteAccount(t *testing.T) {

	Convey("Testing the account 'get' command", t, func() {
		out = testFile()
		command := &remAccGet{}

		Convey("Given a gateway with 1 remote account", func() {
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
			commandLine.Account.Remote.Args.Partner = partner.Name

			account := &model.RemoteAccount{
				Login:         "login",
				Password:      []byte("password"),
				RemoteAgentID: partner.ID,
			}
			So(db.Create(account), ShouldBeNil)

			send := &model.Rule{Name: "send", IsSend: true, Path: "send_path"}
			So(db.Create(send), ShouldBeNil)
			receive := &model.Rule{Name: "receive", IsSend: false, Path: "rcv_path"}
			So(db.Create(receive), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Create(sendAll), ShouldBeNil)

			sAccess := &model.RuleAccess{RuleID: send.ID,
				ObjectType: account.TableName(), ObjectID: account.ID}
			So(db.Create(sAccess), ShouldBeNil)
			rAccess := &model.RuleAccess{RuleID: receive.ID,
				ObjectType: account.TableName(), ObjectID: account.ID}
			So(db.Create(rAccess), ShouldBeNil)

			Convey("Given a valid account name", func() {
				args := []string{account.Login}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the account's info", func() {
						rules := &models.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						a := rest.FromRemoteAccount(account, rules)
						So(getOutput(), ShouldEqual, accInfoString(a))
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+
							partner.Name)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				commandLine.Account.Remote.Args.Partner = "toto"
				args := []string{account.Login}

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

func TestAddRemoteAccount(t *testing.T) {

	Convey("Testing the account 'add' command", t, func() {
		out = testFile()
		command := &remAccAdd{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)
			commandLine.Account.Remote.Args.Partner = partner.Name

			Convey("Given valid flags", func() {
				args := []string{"-l", "login", "-p", "password"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account was added", func() {
						So(getOutput(), ShouldEqual, "The account login "+
							"was successfully added.\n")
					})

					Convey("Then the new partner should have been added", func() {
						account := &model.RemoteAccount{
							Login:         command.Login,
							RemoteAgentID: partner.ID,
						}
						So(db.Get(account), ShouldBeNil)

						clearPwd, err := model.DecryptPassword(account.Password)
						So(err, ShouldBeNil)
						So(string(clearPwd), ShouldEqual, command.Password)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{"-l", "login", "-p", "password"}
				commandLine.Account.Remote.Args.Partner = "toto"

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

func TestDeleteRemoteAccount(t *testing.T) {

	Convey("Testing the account 'delete' command", t, func() {
		out = testFile()
		command := &remAccDelete{}

		Convey("Given a gateway with 1 remote account", func() {
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
			commandLine.Account.Remote.Args.Partner = partner.Name

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			Convey("Given a valid account name", func() {
				args := []string{account.Login}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account was deleted", func() {
						So(getOutput(), ShouldEqual, "The account "+account.Login+
							" was successfully deleted.\n")
					})

					Convey("Then the account should have been removed", func() {
						var accs []model.RemoteAccount
						So(db.Select(&accs, nil), ShouldBeNil)
						So(accs, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+
							partner.Name)
					})

					Convey("Then the account should still exist", func() {
						So(db.Get(account), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{account.Login}
				commandLine.Account.Remote.Args.Partner = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the account should still exist", func() {
						So(db.Get(account), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestUpdateRemoteAccount(t *testing.T) {

	Convey("Testing the account 'delete' command", t, func() {
		out = testFile()
		command := &remAccUpdate{}

		Convey("Given a gateway with 1 remote account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)
			commandLine.Account.Remote.Args.Partner = partner.Name

			oldPwd := []byte("password")
			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "login",
				Password:      oldPwd,
			}
			So(db.Create(account), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-l", "new_login", "-p", "new_password", account.Login}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the "+
						"account was updated", func() {
						So(getOutput(), ShouldEqual, "The account new_login"+
							" was successfully updated.\n")
					})

					Convey("Then the account should have been updated", func() {
						var accs []model.RemoteAccount
						So(db.Select(&accs, nil), ShouldBeNil)
						So(len(accs), ShouldEqual, 1)

						clearPwd, err := model.DecryptPassword(accs[0].Password)
						So(err, ShouldBeNil)
						So(string(clearPwd), ShouldEqual, "new_password")

					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"-l", "new_login", "-p", "new_password", "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+
							partner.Name)
					})

					Convey("Then the account should stay unchanged", func() {
						check := &model.RemoteAccount{
							ID:            account.ID,
							Login:         account.Login,
							RemoteAgentID: account.RemoteAgentID,
						}
						So(db.Get(check), ShouldBeNil)

						clearPwd, err := model.DecryptPassword(check.Password)
						So(err, ShouldBeNil)
						So(string(clearPwd), ShouldEqual, string(oldPwd))
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{"-l", "new_login", "-p", "new_password", account.Login}
				commandLine.Account.Remote.Args.Partner = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the account should stay unchanged", func() {
						check := &model.RemoteAccount{
							ID:            account.ID,
							Login:         account.Login,
							RemoteAgentID: account.RemoteAgentID,
						}
						So(db.Get(check), ShouldBeNil)

						clearPwd, err := model.DecryptPassword(check.Password)
						So(err, ShouldBeNil)
						So(string(clearPwd), ShouldEqual, string(oldPwd))
					})
				})
			})
		})
	})
}

func TestListRemoteAccount(t *testing.T) {

	Convey("Testing the account 'list' command", t, func() {
		out = testFile()
		command := &remAccList{}

		Convey("Given a gateway with 2 remote accounts", func() {
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
			commandLine.Account.Remote.Args.Partner = partner1.Name

			partner2 := &model.RemoteAgent{
				Name:        "partner2",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Create(partner2), ShouldBeNil)

			account1 := &model.RemoteAccount{
				RemoteAgentID: partner1.ID,
				Login:         "account1",
				Password:      []byte("password"),
			}
			So(db.Create(account1), ShouldBeNil)

			account2 := &model.RemoteAccount{
				RemoteAgentID: partner2.ID,
				Login:         "account2",
				Password:      []byte("password"),
			}
			So(db.Create(account2), ShouldBeNil)

			account3 := &model.RemoteAccount{
				RemoteAgentID: partner1.ID,
				Login:         "account3",
				Password:      []byte("password"),
			}
			So(db.Create(account3), ShouldBeNil)

			a1 := rest.FromRemoteAccount(account1, &models.AuthorizedRules{})
			a2 := rest.FromRemoteAccount(account2, &models.AuthorizedRules{})
			a3 := rest.FromRemoteAccount(account3, &models.AuthorizedRules{})

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the partner accounts' info", func() {
						So(getOutput(), ShouldEqual, "Accounts of partner '"+partner1.Name+"':\n"+
							accInfoString(a1)+accInfoString(a3))
					})
				})
			})

			Convey("Given a different partner name", func() {
				args := []string{}
				commandLine.Account.Remote.Args.Partner = partner2.Name

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the partner accounts' info", func() {
						So(getOutput(), ShouldEqual, "Accounts of partner '"+partner2.Name+"':\n"+
							accInfoString(a2))
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{}
				commandLine.Account.Remote.Args.Partner = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				args := []string{"-l", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should only display 1 account's info", func() {
						So(getOutput(), ShouldEqual, "Accounts of partner '"+partner1.Name+"':\n"+
							accInfoString(a1))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				args := []string{"-o", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display all but the 1st account's info", func() {
						So(getOutput(), ShouldEqual, "Accounts of partner '"+partner1.Name+"':\n"+
							accInfoString(a3))
					})
				})
			})

			Convey("Given 'sort' parameter of 'login-'", func() {
				args := []string{"-s", "login-"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the accounts' info in reverse", func() {
						So(getOutput(), ShouldEqual, "Accounts of partner '"+partner1.Name+"':\n"+
							accInfoString(a3)+accInfoString(a1))
					})
				})
			})
		})
	})
}

func TestAuthorizeRemoteAccount(t *testing.T) {

	Convey("Testing the remote account 'authorize' command", t, func() {
		out = testFile()
		command := &remAccAuthorize{}

		Convey("Given a gateway with 1 remote account and 1 rule", func() {
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

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid partner, account & rule names", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account can use the rule", func() {
						So(getOutput(), ShouldEqual, "Usage of the "+direction(rule)+
							" rule '"+rule.Name+"' is now restricted.\nThe remote account "+
							account.Login+" is now allowed to use the "+direction(rule)+
							" rule "+rule.Name+" for transfers.\n")
					})

					Convey("Then the permission should have been added", func() {
						access := &model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   account.ID,
							ObjectType: account.TableName(),
						}
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				commandLine.Account.Remote.Args.Partner = "toto"
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						var a []model.RuleAccess
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{account.Login, "toto", direction(rule)}

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

			Convey("Given an invalid account name", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
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

func TestRevokeRemoteAccount(t *testing.T) {

	Convey("Testing the remote account 'revoke' command", t, func() {
		out = testFile()
		command := &remAccRevoke{}

		Convey("Given a gateway with 1 remote account and 1 rule", func() {
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

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   account.ID,
				ObjectType: account.TableName(),
			}
			So(db.Create(access), ShouldBeNil)

			Convey("Given a valid partner & rule names", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner cannot use the rule", func() {
						So(getOutput(), ShouldEqual, "The remote account "+account.Login+
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

			Convey("Given an invalid partner name", func() {
				commandLine.Account.Remote.Args.Partner = "toto"
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "partner 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{account.Login, "toto", direction(rule)}

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

			Convey("Given an invalid account name", func() {
				commandLine.Account.Remote.Args.Partner = partner.Name
				args := []string{"toto", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
					})

					Convey("Then the permission should NOT have been added", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})
		})
	})
}
