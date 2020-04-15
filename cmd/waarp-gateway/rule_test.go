package main

import (
	"fmt"
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

func ruleInfoString(r *rest.OutRule) string {
	way := "RECEIVE"
	if r.IsSend {
		way = "SEND"
	}

	servers := strings.Join(r.Authorized.LocalServers, ", ")
	partners := strings.Join(r.Authorized.RemotePartners, ", ")
	la := []string{}
	for server, accounts := range r.Authorized.LocalAccounts {
		for _, account := range accounts {
			la = append(la, fmt.Sprint(server, ".", account))
		}
	}
	ra := []string{}
	for partner, accounts := range r.Authorized.RemoteAccounts {
		for _, account := range accounts {
			ra = append(ra, fmt.Sprint(partner, ".", account))
		}
	}
	locAcc := strings.Join(la, ", ")
	remAcc := strings.Join(ra, ", ")

	rv := "● Rule " + r.Name + " (" + way + ")\n" +
		"   Comment: " + r.Comment + "\n" +
		"      Path: " + r.Path + "\n" +
		"    InPath: " + r.InPath + "\n" +
		"   OutPath: " + r.OutPath + "\n" +
		"   Authorized agents\n" +
		"   ├─         Servers: " + servers + "\n" +
		"   ├─        Partners: " + partners + "\n" +
		"   ├─ Server accounts: " + locAcc + "\n" +
		"   └─Partner accounts: " + remAcc + "\n"

	return rv
}

func TestDisplayRule(t *testing.T) {

	Convey("Given a rule entry", t, func() {
		out = testFile()

		rule := &rest.OutRule{
			Name:    "rule_name",
			Comment: "this is a comment",
			IsSend:  true,
			Path:    "rule/path",
			InPath:  "/rule/in_path",
			OutPath: "/rule/out_path",
			Authorized: rest.RuleAccess{
				LocalServers:   []string{"server1", "server2"},
				RemotePartners: []string{"partner1", "partner2"},
				LocalAccounts:  map[string][]string{"server3": {"account1", "account2"}},
				RemoteAccounts: map[string][]string{"partner3": {"account3", "account4"}},
			},
		}
		Convey("When calling the `displayRule` function", func() {
			w := getColorable()
			displayRule(w, rule)

			Convey("Then it should display the rule's info correctly", func() {
				So(getOutput(), ShouldEqual, ruleInfoString(rule))
			})
		})
	})
}

func TestGetRule(t *testing.T) {

	Convey("Testing the rule 'get' command", t, func() {
		out = testFile()
		command := &ruleGet{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			rule := &model.Rule{
				Name:    "rule_name",
				Comment: "this is a test rule",
				IsSend:  false,
				Path:    "/test/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid rule name", func() {
				args := []string{rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the rule's info", func() {
						r := rest.FromRule(rule, &rest.RuleAccess{})
						So(getOutput(), ShouldEqual, ruleInfoString(r))
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "rule 'toto' not found")
					})
				})
			})
		})
	})
}

func TestAddRule(t *testing.T) {

	Convey("Testing the rule 'add' command", t, func() {
		out = testFile()
		command := &ruleAdd{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			existing := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given valid parameters", func() {
				args := []string{"-n", "new_rule", "-c", "new_rule comment",
					"-d", "RECEIVE", "-p", "/new/rule/path"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the rule was added", func() {
						So(getOutput(), ShouldEqual, "The rule "+command.Name+
							" was successfully added.\n")
					})

					Convey("Then the new rule should have been added", func() {
						rule := &model.Rule{
							Name:    command.Name,
							Comment: command.Comment,
							IsSend:  command.Direction == "SEND",
							Path:    command.Path,
						}
						So(db.Get(rule), ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule's name already exist", func() {
				args := []string{"-n", existing.Name, "-c", "new_rule comment",
					"-d", "RECEIVE", "-p", "/new/rule/path"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "a rule named '"+existing.Name+
							"' with send = "+fmt.Sprint(command.Direction == "SEND")+
							" already exist")
					})

					Convey("Then the new rule should not have been added", func() {
						rule := &model.Rule{
							Comment: command.Comment,
							IsSend:  command.Direction == "SEND",
							Path:    command.Path,
						}
						exists, err := db.Exists(rule)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})

					Convey("Then the old rule should still exist", func() {
						So(db.Get(existing), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestDeleteRule(t *testing.T) {

	Convey("Testing the rule 'delete' command", t, func() {
		out = testFile()
		command := &ruleDelete{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/existing/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid rule name", func() {
				args := []string{rule.Name}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the rule was deleted", func() {
						So(getOutput(), ShouldEqual, "The rule "+rule.Name+
							" was successfully deleted.\n")
					})

					Convey("Then the rule should have been removed", func() {
						exists, err := db.Exists(rule)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "rule 'toto' not found")
					})

					Convey("Then the rule should still exist", func() {
						So(db.Get(rule), ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestListRules(t *testing.T) {

	Convey("Testing the rule 'list' command", t, func() {
		out = testFile()
		command := &ruleList{}

		Convey("Given a gateway with 2 rules", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			receive := &model.Rule{
				Name:    "receive",
				Comment: "receive comment",
				IsSend:  false,
				Path:    "/receive/path",
				InPath:  "/receive/in_path",
				OutPath: "/receive/out_path",
			}
			So(db.Create(receive), ShouldBeNil)

			send := &model.Rule{
				Name:    "send",
				Comment: "send comment",
				IsSend:  true,
				Path:    "/send/path",
				InPath:  "/send/in_path",
				OutPath: "/send/out_path",
			}
			So(db.Create(send), ShouldBeNil)

			rcv := rest.FromRule(receive, &rest.RuleAccess{})
			snd := rest.FromRule(send, &rest.RuleAccess{})

			Convey("Given no parameters", func() {
				args := []string{}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the rule' info", func() {
						So(getOutput(), ShouldEqual, "Rules:\n"+
							ruleInfoString(rcv)+ruleInfoString(snd))
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				args := []string{"-l", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display only 1 rule's info", func() {
						So(getOutput(), ShouldEqual, "Rules:\n"+
							ruleInfoString(rcv))
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				args := []string{"-o", "1"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should NOT display the 1st rule", func() {
						So(getOutput(), ShouldEqual, "Rules:\n"+
							ruleInfoString(snd))
					})
				})
			})

			Convey("Given an 'sort' parameter of 'name-'", func() {
				args := []string{"-s", "name-"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the rules' info in reverse", func() {
						So(getOutput(), ShouldEqual, "Rules:\n"+
							ruleInfoString(snd)+ruleInfoString(rcv))
					})
				})
			})
		})
	})
}
