package wg

import (
	"encoding/json"
	"fmt"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func direction(r *model.Rule) string {
	if r.IsSend {
		return directionSend
	}

	return directionRecv
}

func ruleInfoString(r *api.OutRule) string {
	way := directionRecv
	if r.IsSend {
		way = directionSend
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
		"    Comment:                " + r.Comment + "\n" +
		"    Path:                   " + r.Path + "\n" +
		"    Local directory:        " + r.LocalDir + "\n" +
		"    Remote directory:       " + r.RemoteDir + "\n" +
		"    Temp receive directory: " + r.TmpLocalRcvDir + "\n" +
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
			Name:           "rule_name",
			Comment:        "this is a comment",
			IsSend:         true,
			Path:           "rule/path",
			LocalDir:       "/rule/local",
			RemoteDir:      "/rule/remote",
			TmpLocalRcvDir: "/rule/tmp",
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

		Convey("Given a gateway with 1 rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:           "rule_name",
				Comment:        "this is a test rule",
				IsSend:         false,
				Path:           "/rule",
				LocalDir:       "/rule/local",
				RemoteDir:      "/rule/remote",
				TmpLocalRcvDir: "/rule/tmp",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

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
						So(err, ShouldBeError, "receive rule 'toto' not found")
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

		Convey("Given a gateway with 1 rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			existing := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing",
			}
			So(db.Insert(existing).Run(), ShouldBeNil)

			Convey("Given valid parameters", func() {
				args := []string{
					"--name", "new_rule", "--comment", "new_rule comment",
					"--direction", "receive", "--path", "new/rule/path",
					"--local-dir", "new/rule/local",
					"--remote-dir", "new/rule/remote",
					"--tmp-dir", "new_rule/tmp",
					"--pre", `{"type":"COPY","args":{"path":"/path/to/copy"}}`,
					"--pre", `{"type":"EXEC","args":{"path":"/path/to/script","args":"{}","delay":"0"}}`,
					"--post", `{"type":"DELETE","args":{}}`,
					"--post", `{"type":"TRANSFER","args":{"file":"/path/to/file","to":"server","as":"account","rule":"rule"}}`,
					"--err", `{"type":"MOVE","args":{"path":"/path/to/move"}}`,
					"--err", `{"type":"RENAME","args":{"path":"/path/to/rename"}}`,
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
						var rules model.Rules
						So(db.Select(&rules).Where("id=?", 2).Run(), ShouldBeNil)
						So(rules, ShouldNotBeEmpty)

						rule := model.Rule{
							ID:             2,
							Name:           "new_rule",
							Comment:        "new_rule comment",
							IsSend:         false,
							Path:           "new/rule/path",
							LocalDir:       utils.ToOSPath("new/rule/local"),
							RemoteDir:      "new/rule/remote",
							TmpLocalRcvDir: utils.ToOSPath("new_rule/tmp"),
						}
						So(rules[0], ShouldResemble, rule)

						Convey("Then the rule's tasks should have been added", func() {
							var tasks model.Tasks
							So(db.Select(&tasks).Run(), ShouldBeNil)

							pre0 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainPre,
								Rank:   0,
								Type:   "COPY",
								Args:   json.RawMessage(`{"path":"/path/to/copy"}`),
							}
							pre1 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainPre,
								Rank:   1,
								Type:   "EXEC",
								Args: json.RawMessage(
									`{"path":"/path/to/script","args":"{}","delay":"0"}`),
							}
							post0 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainPost,
								Rank:   0,
								Type:   "DELETE",
								Args:   json.RawMessage(`{}`),
							}
							post1 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TRANSFER",
								Args: json.RawMessage(`{"file":"/path/to/file",` +
									`"to":"server","as":"account","rule":"rule"}`),
							}
							err0 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainError,
								Rank:   0,
								Type:   "MOVE",
								Args:   json.RawMessage(`{"path":"/path/to/move"}`),
							}
							err1 := model.Task{
								RuleID: rule.ID,
								Chain:  model.ChainError,
								Rank:   1,
								Type:   "RENAME",
								Args:   json.RawMessage(`{"path":"/path/to/rename"}`),
							}

							So(tasks, ShouldContain, pre0)
							So(tasks, ShouldContain, pre1)
							So(tasks, ShouldContain, post0)
							So(tasks, ShouldContain, post1)
							So(tasks, ShouldContain, err0)
							So(tasks, ShouldContain, err1)
						})
					})
				})
			})

			Convey("Given that the rule's name already exist", func() {
				args := []string{
					"--name", existing.Name, "--direction", "receive",
					"--path", "new_rule/path",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "a "+existing.Direction()+
							" rule named '"+existing.Name+"' already exist")
					})

					Convey("Then the rule should NOT have been inserted", func() {
						var rules model.Rules
						So(db.Select(&rules).Run(), ShouldBeNil)
						So(rules, ShouldHaveLength, 1)

						So(rules[0], ShouldResemble, *existing)
					})
				})
			})

			Convey("Given that that one of the task's JSON is invalid", func() {
				args := []string{
					"--name", "new_rule", "--comment", "new_rule comment",
					"--direction", "receive", "--path", "new/rule/path",
					"--pre", `{"type":NOT_A_TYPE,"args":{"path":"/path/to/copy"}}`,
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "invalid pre task: invalid character"+
							" 'N' looking for beginning of value")
					})

					Convey("Then the rule should NOT have been inserted", func() {
						var rules model.Rules
						So(db.Select(&rules).Run(), ShouldBeNil)
						So(rules, ShouldHaveLength, 1)

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

		Convey("Given a gateway with 1 rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule_name",
				IsSend: true,
				Path:   "/existing",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

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
						var rules model.Rules
						So(db.Select(&rules).Run(), ShouldBeNil)
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
						So(err, ShouldBeError, "send rule 'toto' not found")
					})

					Convey("Then the rule should still exist", func() {
						var rules model.Rules
						So(db.Select(&rules).Run(), ShouldBeNil)
						So(rules, ShouldContain, *rule)
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

		Convey("Given a gateway with 2 rules", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			receive := &model.Rule{
				Name:           "receive_rule",
				Comment:        "receive comment",
				IsSend:         false,
				Path:           "/receive",
				LocalDir:       "/receive/local",
				RemoteDir:      "/receive/remote",
				TmpLocalRcvDir: "/receive/tmp",
			}
			So(db.Insert(receive).Run(), ShouldBeNil)

			send := &model.Rule{
				Name:           "send_rule",
				Comment:        "send comment",
				IsSend:         true,
				Path:           "/send",
				LocalDir:       "/send/local",
				RemoteDir:      "/send/remote",
				TmpLocalRcvDir: "/send/tmp",
			}
			So(db.Insert(send).Run(), ShouldBeNil)

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

		Convey("Given a database with a rule", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "/rule",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			Convey("Given multiple accesses to that rule", func() {
				s := &model.LocalAgent{
					Name:        "server",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:2",
				}
				So(db.Insert(p).Run(), ShouldBeNil)
				So(db.Insert(s).Run(), ShouldBeNil)

				la := &model.LocalAccount{
					LocalAgentID: s.ID,
					Login:        "toto",
					PasswordHash: hash("password"),
				}
				ra := &model.RemoteAccount{
					RemoteAgentID: p.ID,
					Login:         "tata",
					Password:      "password",
				}
				So(db.Insert(la).Run(), ShouldBeNil)
				So(db.Insert(ra).Run(), ShouldBeNil)

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
				So(db.Insert(sAcc).Run(), ShouldBeNil)
				So(db.Insert(pAcc).Run(), ShouldBeNil)
				So(db.Insert(laAcc).Run(), ShouldBeNil)
				So(db.Insert(raAcc).Run(), ShouldBeNil)

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
							var res model.RuleAccesses
							So(db.Select(&res).Run(), ShouldBeNil)
							So(res, ShouldBeEmpty)
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
							So(err, ShouldBeError, "send rule 'toto' not found")
						})

						Convey("Then the accesses should still exist", func() {
							var res model.RuleAccesses
							So(db.Select(&res).Run(), ShouldBeNil)
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
							So(err, ShouldBeError)
							So(err.Error(), ShouldContainSubstring, "invalid rule direction 'toto'")
						})

						Convey("Then the accesses should still exist", func() {
							var res model.RuleAccesses
							So(db.Select(&res).Run(), ShouldBeNil)
							So(len(res), ShouldEqual, 4)
						})
					})
				})
			})
		})
	})
}
