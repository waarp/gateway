package wg

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func accInfoString(a *api.OutAccount) string {
	return "● Account " + a.Login + "\n" +
		"    Authorized rules\n" +
		"    ├─  Sending: " + strings.Join(a.AuthorizedRules.Sending, ", ") + "\n" +
		"    └─Reception: " + strings.Join(a.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetLocalAccount(t *testing.T) {
	Convey("Testing the local account 'get' command", t, func() {
		out = testFile()
		command := &locAccGet{}

		Convey("Given a gateway with 1 local account", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			account := &model.LocalAccount{
				Login:        "toto",
				PasswordHash: hash("sesame"),
				LocalAgentID: server.ID,
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			send := &model.Rule{Name: "send_rule", IsSend: true, Path: "send_path"}
			So(db.Insert(send).Run(), ShouldBeNil)
			receive := &model.Rule{Name: "recv_rule", IsSend: false, Path: "rcv_path"}
			So(db.Insert(receive).Run(), ShouldBeNil)
			sendAll := &model.Rule{Name: "send_all", IsSend: true, Path: "send_all_path"}
			So(db.Insert(sendAll).Run(), ShouldBeNil)

			sAccess := &model.RuleAccess{
				RuleID:     send.ID,
				ObjectType: account.TableName(), ObjectID: account.ID,
			}
			So(db.Insert(sAccess).Run(), ShouldBeNil)
			rAccess := &model.RuleAccess{
				RuleID:     receive.ID,
				ObjectType: account.TableName(), ObjectID: account.ID,
			}
			So(db.Insert(rAccess).Run(), ShouldBeNil)

			Convey("Given a valid account name", func() {
				args := []string{account.Login}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the account's info", func() {
						rules := &api.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						a := rest.FromLocalAccount(account, rules)
						So(getOutput(), ShouldEqual, accInfoString(a))
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"tata"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for server "+
							server.Name)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				commandLine.Account.Local.Args.Server = "toto"
				args := []string{account.Login}

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

func TestAddLocalAccount(t *testing.T) {
	Convey("Testing the local account 'add' command", t, func() {
		out = testFile()
		command := &locAccAdd{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			Convey("Given valid flags", func() {
				args := []string{"-l", "toto", "-p", "sesame"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The account toto "+
							"was successfully added.\n")
					})

					Convey("Then the new account should have been added", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldNotBeEmpty)

						So(bcrypt.CompareHashAndPassword([]byte(accounts[0].PasswordHash),
							[]byte(command.Password)), ShouldBeNil)
						exp := model.LocalAccount{
							ID:           1,
							LocalAgentID: server.ID,
							Login:        command.Login,
							PasswordHash: accounts[0].PasswordHash,
						}
						So(accounts, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"-l", "toto", "-p", "sesame"}
				commandLine.Account.Local.Args.Server = "toto"

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

func TestDeleteLocalAccount(t *testing.T) {
	Convey("Testing the local account 'delete' command", t, func() {
		out = testFile()
		command := &locAccDelete{}

		Convey("Given a gateway with 1 local account", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			Convey("Given a valid account name", func() {
				args := []string{account.Login}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account "+
						"was deleted", func() {
						So(getOutput(), ShouldEqual, "The account "+account.Login+
							" was successfully deleted.\n")
					})

					Convey("Then the account should have been removed", func() {
						var accounts model.RemoteAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"tata"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for server "+
							server.Name)
					})

					Convey("Then the account should still exist", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldContain, *account)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{account.Login}
				commandLine.Account.Local.Args.Server = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the account should still exist", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldContain, *account)
					})
				})
			})
		})
	})
}

func TestUpdateLocalAccount(t *testing.T) {
	Convey("Testing the local account 'update' command", t, func() {
		out = testFile()
		command := &locAccUpdate{}

		Convey("Given a gateway with 1 local account", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server := &model.LocalAgent{
				Name:     "parent",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(server).Run(), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

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
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldNotBeEmpty)

						So(bcrypt.CompareHashAndPassword([]byte(accounts[0].PasswordHash),
							[]byte("new_password")), ShouldBeNil)
						exp := model.LocalAccount{
							ID:           account.ID,
							LocalAgentID: account.LocalAgentID,
							Login:        "new_login",
							PasswordHash: accounts[0].PasswordHash,
						}
						So(accounts, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{"-l", "new_login", "-p", "new_password", "tata"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for server "+
							server.Name)
					})

					Convey("Then the account should stay unchanged", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldContain, *account)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"-l", "new_login", "-p", "new_password", account.Login}
				commandLine.Account.Local.Args.Server = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the account should stay unchanged", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldContain, *account)
					})
				})
			})
		})
	})
}

func TestListLocalAccount(t *testing.T) {
	Convey("Testing the local account 'list' command", t, func() {
		out = testFile()
		command := &locAccList{}

		Convey("Given a gateway with 2 local accounts", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			server1 := &model.LocalAgent{
				Name:        "server1",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(server1).Run(), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server1.Name

			server2 := &model.LocalAgent{
				Name:        "server2",
				Protocol:    testProto1,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			So(db.Insert(server2).Run(), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: server1.ID,
				Login:        "account1",
				PasswordHash: hash("password1"),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: server2.ID,
				Login:        "account2",
				PasswordHash: hash("password2"),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			account3 := &model.LocalAccount{
				LocalAgentID: server1.ID,
				Login:        "account3",
				PasswordHash: hash("password3"),
			}
			So(db.Insert(account3).Run(), ShouldBeNil)

			a1 := rest.FromLocalAccount(account1, &api.AuthorizedRules{})
			a2 := rest.FromLocalAccount(account2, &api.AuthorizedRules{})
			a3 := rest.FromLocalAccount(account3, &api.AuthorizedRules{})

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the server accounts' info", func() {
						So(getOutput(), ShouldEqual, "Accounts of server '"+server1.Name+"':\n"+
							accInfoString(a1)+accInfoString(a3))
					})
				})
			})

			Convey("Given a different server name", func() {
				args := []string{}
				commandLine.Account.Local.Args.Server = server2.Name

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the server accounts' info", func() {
						So(getOutput(), ShouldEqual, "Accounts of server '"+server2.Name+"':\n"+
							accInfoString(a2))
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{}
				commandLine.Account.Local.Args.Server = "toto"

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
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
						So(getOutput(), ShouldEqual, "Accounts of server '"+server1.Name+"':\n"+
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
						So(getOutput(), ShouldEqual, "Accounts of server '"+server1.Name+"':\n"+
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
						So(getOutput(), ShouldEqual, "Accounts of server '"+server1.Name+"':\n"+
							accInfoString(a3)+accInfoString(a1))
					})
				})
			})
		})
	})
}

func TestAuthorizeLocalAccount(t *testing.T) {
	Convey("Testing the local account 'authorize' command", t, func() {
		out = testFile()
		command := &locAccAuthorize{}

		Convey("Given a gateway with 1 local account and 1 rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
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

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			Convey("Given a valid partner, account & rule names", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account can use the rule", func() {
						So(getOutput(), ShouldEqual, "Usage of the "+direction(rule)+
							" rule '"+rule.Name+"' is now restricted.\nThe local account "+
							account.Login+" is now allowed to use the "+direction(rule)+
							" rule "+rule.Name+" for transfers.\n")
					})

					Convey("Then the permission should have been added", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)

						exp := model.RuleAccess{
							RuleID:     rule.ID,
							ObjectID:   account.ID,
							ObjectType: account.TableName(),
						}
						So(accesses, ShouldContain, exp)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				commandLine.Account.Local.Args.Server = "toto"
				args := []string{account.Login, rule.Name, direction(rule)}

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

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, "toto", direction(rule)}

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

			Convey("Given an invalid account name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{"tata", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for server "+server.Name)
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

func TestRevokeLocalAccount(t *testing.T) {
	Convey("Testing the local account 'revoke' command", t, func() {
		out = testFile()
		command := &locAccRevoke{}

		Convey("Given a gateway with 1 local account and 1 rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
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

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
				PasswordHash: hash("sesame"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   account.ID,
				ObjectType: account.TableName(),
			}
			So(db.Insert(access).Run(), ShouldBeNil)

			Convey("Given a valid partner & rule names", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner cannot use the rule", func() {
						So(getOutput(), ShouldEqual, "The local account "+account.Login+
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

			Convey("Given an invalid server name", func() {
				commandLine.Account.Local.Args.Server = "toto"
				args := []string{account.Login, rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						var accesses model.RuleAccesses
						So(db.Select(&accesses).Run(), ShouldBeNil)
						So(accesses, ShouldContain, *access)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, "toto", direction(rule)}

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

			Convey("Given an invalid account name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{"tata", rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for server "+server.Name)
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
