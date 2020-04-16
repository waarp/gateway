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
	"golang.org/x/crypto/bcrypt"
)

func accInfoString(a *rest.OutAccount) string {
	return "● Account " + a.Login + "\n" +
		"   Authorized rules\n" +
		"   ├─  Sending: " + strings.Join(a.AuthorizedRules.Sending, ", ") + "\n" +
		"   └─Reception: " + strings.Join(a.AuthorizedRules.Reception, ", ") + "\n"
}

func TestGetLocalAccount(t *testing.T) {

	Convey("Testing the local account 'get' command", t, func() {
		out = testFile()
		command := &locAccGet{}

		Convey("Given a gateway with 1 local account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			account := &model.LocalAccount{
				Login:        "login",
				Password:     []byte("password"),
				LocalAgentID: server.ID,
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
						rules := &rest.AuthorizedRules{
							Sending:   []string{send.Name, sendAll.Name},
							Reception: []string{receive.Name},
						}
						a := rest.FromLocalAccount(account, rules)
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
						So(err, ShouldBeError, "no account 'toto' found for server "+
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

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			Convey("Given valid flags", func() {
				args := []string{"-l", "login", "-p", "password"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the server was added", func() {
						So(getOutput(), ShouldEqual, "The account login "+
							"was successfully added.\n")
					})

					Convey("Then the new account should have been added", func() {
						account := &model.LocalAccount{
							Login:        command.Login,
							LocalAgentID: server.ID,
						}
						So(db.Get(account), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(account.Password,
							[]byte(command.Password)), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				args := []string{"-l", "login", "-p", "password"}
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

		Convey("Given a gateway with 1 local account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "parent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "login",
				Password:     []byte("password"),
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
						exists, err := db.Exists(&model.LocalAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
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
						So(err, ShouldBeError, "no account 'toto' found for server "+
							server.Name)
					})

					Convey("Then the account should still exist", func() {
						exists, err := db.Exists(&model.LocalAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
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
						exists, err := db.Exists(&model.LocalAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
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

		Convey("Given a gateway with 1 local account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "par²ent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server.Name

			oldPwd := []byte("password")
			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "login",
				Password:     oldPwd,
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

					Convey("Then the old values should have been removed", func() {
						exists, err := db.Exists(account)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})

					Convey("Then the new values should have been added", func() {
						newAccount := &model.LocalAccount{
							ID:           account.ID,
							Login:        command.Login,
							LocalAgentID: account.LocalAgentID,
						}
						So(db.Get(newAccount), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(newAccount.Password,
							[]byte(command.Password)), ShouldBeNil)
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
						So(err, ShouldBeError, "no account 'toto' found for server "+
							server.Name)
					})

					Convey("Then the account should stay unchanged", func() {
						check := &model.LocalAccount{
							ID:           account.ID,
							Login:        account.Login,
							LocalAgentID: account.LocalAgentID,
						}
						So(db.Get(check), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(check.Password,
							oldPwd), ShouldBeNil)
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
						check := &model.LocalAccount{
							ID:           account.ID,
							Login:        account.Login,
							LocalAgentID: account.LocalAgentID,
						}
						So(db.Get(check), ShouldBeNil)
						So(bcrypt.CompareHashAndPassword(check.Password,
							oldPwd), ShouldBeNil)
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

		Convey("Given a gateway with 2 local accounts", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server1 := &model.LocalAgent{
				Name:        "server1",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server1), ShouldBeNil)
			commandLine.Account.Local.Args.Server = server1.Name

			server2 := &model.LocalAgent{
				Name:        "server2",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server2), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: server1.ID,
				Login:        "account1",
				Password:     []byte("password"),
			}
			So(db.Create(account1), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: server2.ID,
				Login:        "account2",
				Password:     []byte("password"),
			}
			So(db.Create(account2), ShouldBeNil)

			account3 := &model.LocalAccount{
				LocalAgentID: server1.ID,
				Login:        "account3",
				Password:     []byte("password"),
			}
			So(db.Create(account3), ShouldBeNil)

			a1 := rest.FromLocalAccount(account1, &rest.AuthorizedRules{})
			a2 := rest.FromLocalAccount(account2, &rest.AuthorizedRules{})
			a3 := rest.FromLocalAccount(account3, &rest.AuthorizedRules{})

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

		Convey("Given a gateway with 1 local account and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "login",
				Password:     []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid partner, account & rule names", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the account can use the rule", func() {
						So(getOutput(), ShouldEqual, "Usage of the rule '"+rule.Name+
							"' is now restricted.\nThe local account "+account.Login+
							" is now allowed to use the rule "+rule.Name+" for transfers.\n")
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

			Convey("Given an invalid server name", func() {
				commandLine.Account.Local.Args.Server = "toto"
				args := []string{account.Login, rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						a := []model.RuleAccess{}
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "rule 'toto' not found")
					})

					Convey("Then the permission should NOT have been added", func() {
						a := []model.RuleAccess{}
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid account name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{"toto", rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
					})

					Convey("Then the permission should NOT have been added", func() {
						a := []model.RuleAccess{}
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
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

		Convey("Given a gateway with 1 local account and 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			server := &model.LocalAgent{
				Name:        "server",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(server), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "login",
				Password:     []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			access := &model.RuleAccess{
				RuleID:     rule.ID,
				ObjectID:   account.ID,
				ObjectType: account.TableName(),
			}
			So(db.Create(access), ShouldBeNil)

			Convey("Given a valid partner & rule names", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the partner cannot use the rule", func() {
						So(getOutput(), ShouldEqual, "The local account "+account.Login+
							" is no longer allowed to use the rule "+rule.Name+
							" for transfers.\nUsage of the rule '"+rule.Name+
							"' is now unrestricted.\n")
					})

					Convey("Then the permission should have been removed", func() {
						a := []model.RuleAccess{}
						So(db.Select(&a, nil), ShouldBeNil)
						So(a, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid server name", func() {
				commandLine.Account.Local.Args.Server = "toto"
				args := []string{account.Login, rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "server 'toto' not found")
					})

					Convey("Then the permission should NOT have been removed", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{account.Login, "toto"}

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
				commandLine.Account.Local.Args.Server = server.Name
				args := []string{"toto", rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then is should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
					})

					Convey("Then the permission should NOT have been added", func() {
						So(db.Get(access), ShouldBeNil)
					})
				})
			})
		})
	})
}
