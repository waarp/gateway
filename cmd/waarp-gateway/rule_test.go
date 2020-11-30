package main

import (
	"encoding/json"
	"fmt"
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

func direction(r *model.Rule) string {
	if r.IsSend {
		return "send"
	}
	return "receive"
}

func ruleInfoString(r *api.OutRule) string {
	way := "receive"
	if r.IsSend {
		way = "send"
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

	taskStr := func(tasks []api.Task) string {
		str := ""
		for i, t := range tasks {
			prefix := "    ├─Command "
			if i == len(tasks)-1 {
				prefix = "    └─Command "
			}
			str += prefix + t.Type + " with args: " + string(t.Args) + "\n"
		}
		return str
	}

	rv := "● Rule " + r.Name + " (" + way + ")\n" +
		"    Comment:        " + r.Comment + "\n" +
		"    Path:           " + r.Path + "\n" +
		"    In directory:   " + r.InPath + "\n" +
		"    Out directory:  " + r.OutPath + "\n" +
		"    Work directory: " + r.WorkPath + "\n" +
		"    Pre tasks:\n" + taskStr(r.PreTasks) +
		"    Post tasks:\n" + taskStr(r.PostTasks) +
		"    Error tasks:\n" + taskStr(r.ErrorTasks) +
		"    Authorized agents:\n" +
		"    ├─Servers:          " + servers + "\n" +
		"    ├─Partners:         " + partners + "\n" +
		"    ├─Server accounts:  " + locAcc + "\n" +
		"    └─Partner accounts: " + remAcc + "\n"

	return rv
}

func TestDisplayRule(t *testing.T) {

	Convey("Given a rule entry", t, func() {
		out = testFile()

		rule := &api.OutRule{
			Name:    "rule_name",
			Comment: "this is a comment",
			IsSend:  true,
			Path:    "rule/path",
			InPath:  "rule/in_path",
			OutPath: "rule/out_path",
			Authorized: &api.RuleAccess{
				LocalServers:   []string{"server1", "server2"},
				RemotePartners: []string{"partner1", "partner2"},
				LocalAccounts:  map[string][]string{"server3": {"account1", "account2"}},
				RemoteAccounts: map[string][]string{"partner3": {"account3", "account4"}},
			},
			PreTasks: []api.Task{{
				Type: "COPY",
				Args: json.RawMessage(`{"path":"/path/to/copy"}`),
			}, {
				Type: "EXEC",
				Args: json.RawMessage(`{"path":"/path/to/script","args":"{}","delay":"0"}`),
			}},
			PostTasks: []api.Task{{
				Type: "DELETE",
				Args: json.RawMessage("{}"),
			}, {
				Type: "TRANSFER",
				Args: json.RawMessage(`{"file":"/path/to/file","to":"server","as":"account","rule":"rule"}`),
			}},
			ErrorTasks: []api.Task{{
				Type: "MOVE",
				Args: json.RawMessage(`{"path":"/path/to/move"}`),
			}, {
				Type: "RENAME",
				Args: json.RawMessage(`{"path":"/path/to/rename"}`),
			}},
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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:     "rule_name",
				Comment:  "this is a test rule",
				IsSend:   false,
				Path:     "test/rule/path",
				InPath:   "test/rule/in",
				OutPath:  "test/rule/out",
				WorkPath: "test/rule/work",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid rule name", func() {
				args := []string{rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then it should display the rule's info", func() {
						r, err := rest.FromRule(db, rule)
						So(err, ShouldBeNil)
						So(getOutput(), ShouldEqual, ruleInfoString(r))
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"toto", direction(rule)}

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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			existing := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "existing/rule/path",
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given valid parameters", func() {
				args := []string{"-n", "new_rule", "-c", "new_rule comment",
					"-d", "receive", "--path=/new/rule/path", "--out_path=/out/path",
					"--in_path=/in/path", "--work_path=/work/path",
					`--pre={"type":"COPY","args":{"path":"/path/to/copy"}}`,
					`--pre={"type":"EXEC","args":{"path":"/path/to/script","args":"{}","delay":"0"}}`,
					`--post={"type":"DELETE","args":{}}`,
					`--post={"type":"TRANSFER","args":{"file":"/path/to/file","to":"server","as":"account","rule":"rule"}}`,
					`--err={"type":"MOVE","args":{"path":"/path/to/move"}}`,
					`--err={"type":"RENAME","args":{"path":"/path/to/rename"}}`,
				}

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
							Name:     command.Name,
							Comment:  *command.Comment,
							IsSend:   command.Direction == "send",
							Path:     command.Path,
							InPath:   *command.InPath,
							OutPath:  *command.OutPath,
							WorkPath: *command.WorkPath,
						}
						So(db.Get(rule), ShouldBeNil)

						pre0 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainPre,
							Rank:   0,
							Type:   "COPY",
							Args:   json.RawMessage(`{"path":"/path/to/copy"}`),
						}
						So(db.Get(pre0), ShouldBeNil)
						pre1 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainPre,
							Rank:   1,
							Type:   "EXEC",
							Args: json.RawMessage(
								`{"path":"/path/to/script","args":"{}","delay":"0"}`),
						}
						So(db.Get(pre1), ShouldBeNil)

						post0 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainPost,
							Rank:   0,
							Type:   "DELETE",
							Args:   json.RawMessage(`{}`),
						}
						So(db.Get(post0), ShouldBeNil)
						post1 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainPost,
							Rank:   1,
							Type:   "TRANSFER",
							Args: json.RawMessage(`{"file":"/path/to/file",` +
								`"to":"server","as":"account","rule":"rule"}`),
						}
						So(db.Get(post1), ShouldBeNil)

						err0 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainError,
							Rank:   0,
							Type:   "MOVE",
							Args:   json.RawMessage(`{"path":"/path/to/move"}`),
						}
						So(db.Get(err0), ShouldBeNil)
						err1 := &model.Task{
							RuleID: rule.ID,
							Chain:  model.ChainError,
							Rank:   1,
							Type:   "RENAME",
							Args:   json.RawMessage(`{"path":"/path/to/rename"}`),
						}
						So(db.Get(err1), ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule's name already exist", func() {
				args := []string{"-n", existing.Name, "-c", "new_rule comment",
					"-d", "receive", "-p", "new/rule/path"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "a rule named '"+existing.Name+
							"' with send = "+fmt.Sprint(command.Direction == "send")+
							" already exist")
					})

					Convey("Then the rule should have been updated", func() {
						var rules []model.Rule
						So(db.Select(&rules, nil), ShouldBeNil)
						So(len(rules), ShouldEqual, 1)
						So(rules[0], ShouldResemble, *existing)
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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "existing/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a valid rule name", func() {
				args := []string{rule.Name, direction(rule)}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the rule was deleted", func() {
						So(getOutput(), ShouldEqual, "The rule "+rule.Name+
							" was successfully deleted.\n")
					})

					Convey("Then the rule should have been removed", func() {
						var rules []model.Rule
						So(db.Select(&rules, nil), ShouldBeNil)
						So(rules, ShouldBeEmpty)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"toto", direction(rule)}

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
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			receive := &model.Rule{
				Name:     "receive",
				Comment:  "receive comment",
				IsSend:   false,
				Path:     "receive/path",
				InPath:   "receive/in_path",
				OutPath:  "receive/out_path",
				WorkPath: "receive/work_path",
			}
			So(db.Create(receive), ShouldBeNil)

			send := &model.Rule{
				Name:     "send",
				Comment:  "send comment",
				IsSend:   true,
				Path:     "send/path",
				InPath:   "send/in_path",
				OutPath:  "send/out_path",
				WorkPath: "send/work_path",
			}
			So(db.Create(send), ShouldBeNil)

			rcv, err := rest.FromRule(db, receive)
			So(err, ShouldBeNil)
			snd, err := rest.FromRule(db, send)
			So(err, ShouldBeNil)

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

func TestRuleAllowAll(t *testing.T) {

	Convey("Testing the rule 'list' command", t, func() {
		out = testFile()
		command := &ruleAllowAll{}

		Convey("Given a database with a rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given multiple accesses to that rule", func() {
				s := &model.LocalAgent{
					Name:        "server",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:2",
				}
				So(db.Create(p), ShouldBeNil)
				So(db.Create(s), ShouldBeNil)

				la := &model.LocalAccount{
					LocalAgentID: s.ID,
					Login:        "toto",
					Password:     []byte("password"),
				}
				ra := &model.RemoteAccount{
					RemoteAgentID: p.ID,
					Login:         "tata",
					Password:      []byte("password"),
				}
				So(db.Create(la), ShouldBeNil)
				So(db.Create(ra), ShouldBeNil)

				sAcc := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   s.ID,
					ObjectType: s.TableName(),
				}
				pAcc := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   p.ID,
					ObjectType: p.TableName(),
				}
				laAcc := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   la.ID,
					ObjectType: la.TableName(),
				}
				raAcc := &model.RuleAccess{
					RuleID:     rule.ID,
					ObjectID:   ra.ID,
					ObjectType: ra.TableName(),
				}
				So(db.Create(sAcc), ShouldBeNil)
				So(db.Create(pAcc), ShouldBeNil)
				So(db.Create(laAcc), ShouldBeNil)
				So(db.Create(raAcc), ShouldBeNil)

				Convey("Given correct command parameters", func() {
					args := []string{rule.Name, direction(rule)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should say that the rule is now unrestricted", func() {
							So(getOutput(), ShouldEqual, "The use of the "+direction(rule)+
								" rule "+rule.Name+" is now unrestricted.\n")
						})

						Convey("Then all accesses should have been removed from the database", func() {
							var res []model.RuleAccess
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 0)
						})
					})
				})

				Convey("Given an incorrect rule name", func() {
					args := []string{"toto", direction(rule)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						err = command.Execute(params)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "rule 'toto' not found")
						})

						Convey("Then the accesses should still exist", func() {
							var res []model.RuleAccess
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 4)
						})
					})
				})

				Convey("Given an incorrect rule direction", func() {
					args := []string{rule.Name, "toto"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						err = command.Execute(params)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "invalid rule direction 'toto'")
						})

						Convey("Then the accesses should still exist", func() {
							var res []model.RuleAccess
							So(db.Select(&res, nil), ShouldBeNil)
							So(len(res), ShouldEqual, 4)
						})
					})
				})
			})
		})
	})
}
