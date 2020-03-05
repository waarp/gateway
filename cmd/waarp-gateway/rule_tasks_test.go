package main

import (
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

func TestChangeRuleTasks(t *testing.T) {

	Convey("Testing the rule tasks 'change' command", t, func() {
		out = testFile()
		command := &ruleTasksChangeCommand{}

		Convey("Given a gateway with 1 rule", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			rule := &model.Rule{
				Name:    "existing rule",
				Comment: "comment about existing rule",
				IsSend:  false,
				Path:    "/existing/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)
			ruleID := fmt.Sprint(rule.ID)

			Convey("When adding a new rule access", func() {

				Convey("Given valid parameters", func() {
					preTasks := []rest.InRuleTask{
						{
							Type: "COPY",
							Args: json.RawMessage("{}"),
						}, {
							Type: "EXEC",
							Args: json.RawMessage("{}"),
						},
					}

					postTasks := []rest.InRuleTask{
						{
							Type: "DELETE",
							Args: json.RawMessage("{}"),
						}, {
							Type: "TRANSFER",
							Args: json.RawMessage("{}"),
						},
					}

					errorTasks := []rest.InRuleTask{
						{
							Type: "MOVE",
							Args: json.RawMessage("{}"),
						}, {
							Type: "RENAME",
							Args: json.RawMessage("{}"),
						},
					}

					pre, err := json.Marshal(preTasks)
					So(err, ShouldBeNil)
					post, err := json.Marshal(postTasks)
					So(err, ShouldBeNil)
					erro, err := json.Marshal(errorTasks)
					So(err, ShouldBeNil)

					command.PreTasks = string(pre)
					command.PostTasks = string(post)
					command.ErrorTasks = string(erro)

					Convey("Given a valid rule ID parameter", func() {
						args := []string{ruleID}

						Convey("When executing the command", func() {
							addr := gw.Listener.Addr().String()
							dsn := "http://admin:admin_password@" + addr
							auth.DSN = dsn

							err := command.Execute(args)

							Convey("Then it should NOT return an error", func() {
								So(err, ShouldBeNil)
							})

							Convey("Then is should display a message saying the tasks "+
								" were added", func() {
								So(getOutput(), ShouldEqual, "The task chains of "+
									"rule n°1 were successfully changed. The rule's "+
									"chains can be consulted at the address: "+
									gw.URL+admin.APIPath+rest.RulesPath+"/"+ruleID+
									rest.RuleTasksPath)
							})

							Convey("Then the new tasks should have been added", func() {
								for _, task := range preTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeTrue)
								}
								for _, task := range postTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeTrue)
								}
								for _, task := range errorTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeTrue)
								}
							})
						})
					})

					Convey("Given an invalid rule ID", func() {
						args := []string{"1000"}

						Convey("When executing the command", func() {
							addr := gw.Listener.Addr().String()
							dsn := "http://admin:admin_password@" + addr
							auth.DSN = dsn

							err := command.Execute(args)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError)
								So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
									addr+admin.APIPath+rest.RulesPath+"/1000"+
									rest.RuleTasksPath+"' does not exist")
							})

							Convey("Then the new tasks should NOT have been added", func() {
								for _, task := range preTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeFalse)
								}
								for _, task := range postTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeFalse)
								}
								for _, task := range errorTasks {
									exists, err := db.Exists(task.ToModel())
									So(err, ShouldBeNil)
									So(exists, ShouldBeFalse)
								}
							})
						})
					})
				})
			})
		})
	})
}

func TestListRuleTasks(t *testing.T) {

	Convey("Testing the rule 'list' command", t, func() {
		out = testFile()
		command := &ruleTasksListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 3 rule tasks", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			rule := &model.Rule{
				Name:    "rule",
				Comment: "rule comment",
				IsSend:  true,
				Path:    "/rule/path",
			}
			So(db.Create(rule), ShouldBeNil)

			pre1 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "COPY",
				Args:   []byte("{}"),
			}
			pre2 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "EXEC",
				Args:   []byte("{}"),
			}
			So(db.Create(pre1), ShouldBeNil)
			So(db.Create(pre2), ShouldBeNil)

			post1 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "DELETE",
				Args:   []byte("{}"),
			}
			post2 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   "TRANSFER",
				Args:   []byte("{}"),
			}
			So(db.Create(post1), ShouldBeNil)
			So(db.Create(post2), ShouldBeNil)

			err1 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainError,
				Rank:   1,
				Type:   "MOVE",
				Args:   []byte("{}"),
			}
			err2 := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainError,
				Rank:   2,
				Type:   "RENAME",
				Args:   []byte("{}"),
			}
			So(db.Create(err1), ShouldBeNil)
			So(db.Create(err2), ShouldBeNil)

			Convey("Given a valid rule ID parameter", func() {
				ruleID := fmt.Sprint(rule.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{ruleID})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the rule accesses", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Pre tasks:\n"+
							"  Command: "+pre2.Type+"\n"+
							"    Arguments: "+string(pre2.Args)+"\n"+
							"  Command: "+pre1.Type+"\n"+
							"    Arguments: "+string(pre1.Args)+"\n"+
							"Post tasks:\n"+
							"  Command: "+post1.Type+"\n"+
							"    Arguments: "+string(post1.Args)+"\n"+
							"  Command: "+post2.Type+"\n"+
							"    Arguments: "+string(post2.Args)+"\n"+
							"Error tasks:\n"+
							"  Command: "+err1.Type+"\n"+
							"    Arguments: "+string(err1.Args)+"\n"+
							"  Command: "+err2.Type+"\n"+
							"    Arguments: "+string(err2.Args)+"\n",
						)
					})
				})
			})

			Convey("Given an invalid rule ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.RulesPath+"/1000"+
							rest.RuleTasksPath+"' does not exist")
					})
				})
			})
		})
	})
}
