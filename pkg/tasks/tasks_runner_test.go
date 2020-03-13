package tasks

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

func TestSetup(t *testing.T) {
	Convey("Given a Task with some replacement variables", t, func() {
		task := &model.Task{
			Type: "DUMMY",
			Args: []byte(`{"rule":"#RULE#", "date":"#DATE#", "hour":"#HOUR#", 
			"path":"#OUTPATH#", "trueFullPath":"#TRUEFULLPATH#",
			"trueFilename":"#TRUEFILENAME#", "fullPath":"#ORIGINALFULLPATH#", 
			"filename":"#ORIGINALFILENAME#", "remoteHost":"#REMOTEHOST#", 
			"localHost":"#LOCALHOST#", "transferID":"#TRANFERID#",
			"requesterHost":"#REQUESTERHOST#", "requestedHost":"#REQUESTEDHOST#",
			"fullTransferID":"#FULLTRANFERID#", "errCode":"#ERRORCODE#",
			"errMsg":"#ERRORMSG#", "errStrCode":"#ERRORSTRCODE#"}`),
		}

		Convey("Given a Processor", func() {
			db := database.GetTestDatabase()

			agent := &model.RemoteAgent{
				Name:        "agent",
				Protocol:    "r66",
				ProtoConfig: []byte("{}"),
			}
			So(db.Create(agent), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "account",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			r := &Processor{
				Db: db,
				Rule: &model.Rule{
					Name:   "Test",
					IsSend: true,
					Path:   "path/to/test",
				},
				Transfer: &model.Transfer{
					ID:        1234,
					IsServer:  false,
					AgentID:   agent.ID,
					AccountID: account.ID,
					Error: model.TransferError{
						Details: `", "bad":1`,
					},
				},
			}

			Convey("When calling the `setup` function", func() {
				res, err := r.setup(task)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then res should contain an entry `rule`", func() {
						val, ok := res["rule"]
						So(ok, ShouldBeTrue)

						Convey("Then res[rule] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Rule.Name)
						})
					})

					Convey("Then res should contain an entry `date`", func() {
						val, ok := res["date"]
						So(ok, ShouldBeTrue)

						Convey("Then res[date] should contain the resolved variable", func() {
							today := time.Now().Format("20060102")
							So(val, ShouldEqual, today)
						})
					})

					Convey("Then res should contain an entry `hour`", func() {
						val, ok := res["hour"]
						So(ok, ShouldBeTrue)

						Convey("Then res[hour] should contain the resolved variable", func() {
							today := time.Now().Format("030405")
							So(val, ShouldEqual, today)
						})
					})

					Convey("Then res should contain an entry `path`", func() {
						val, ok := res["path"]
						So(ok, ShouldBeTrue)

						Convey("Then res[path] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Rule.Path)
						})
					})

					Convey("Then res should contain an entry `trueFullPath`", func() {
						val, ok := res["trueFullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Transfer.SourcePath)
						})
					})

					Convey("Then res should contain an entry `trueFilename`", func() {
						val, ok := res["trueFilename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFilename] should contain the resolved variable", func() {
							So(val, ShouldEqual, filepath.Base(r.Transfer.SourcePath))
						})
					})

					Convey("Then res should contain an entry `fullPath`", func() {
						val, ok := res["fullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[fullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Transfer.SourcePath)
						})
					})

					Convey("Then res should contain an entry `filename`", func() {
						val, ok := res["filename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[filename] should contain the resolved variable", func() {
							So(val, ShouldEqual, filepath.Base(r.Transfer.SourcePath))
						})
					})

					Convey("Then res should contain an entry `remoteHost`", func() {
						val, ok := res["remoteHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[remoteHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, agent.Name)
						})
					})

					Convey("Then res should contain an entry `localHost`", func() {
						val, ok := res["localHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[localHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, account.Login)
						})
					})

					Convey("Then res should contain an entry `transferID`", func() {
						val, ok := res["transferID"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferID] should contain the resolved variable", func() {
							So(val, ShouldEqual, fmt.Sprint(r.Transfer.ID))
						})
					})

					Convey("Then res should contain an entry `requesterHost`", func() {
						val, ok := res["requesterHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[requesterHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, account.Login)
						})
					})

					Convey("Then res should contain an entry `requestedHost`", func() {
						val, ok := res["requestedHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[requestedHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, agent.Name)
						})
					})

					Convey("Then res should contain an entry `fullTransferID`", func() {
						val, ok := res["fullTransferID"]
						So(ok, ShouldBeTrue)

						Convey("Then res[fullTransferID] should contain the resolved variable", func() {
							So(val, ShouldEqual, fmt.Sprintf("%d_%s_%s", r.Transfer.ID,
								account.Login, agent.Name))
						})
					})

					Convey("Then res should contain an entry `errCode`", func() {
						val, ok := res["errCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, string(r.Transfer.Error.Code.R66Code()))
						})
					})

					Convey("Then res should contain an entry `errMsg`", func() {
						val, ok := res["errMsg"]
						So(ok, ShouldBeTrue)

						Convey("Then res[msg] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Transfer.Error.Details)
						})
					})

					Convey("Then res should contain an entry `errStrCode`", func() {
						val, ok := res["errStrCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errStrCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Transfer.Error.Details)
						})
					})
				})
			})
		})
	})
}

func TestGetTasks(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule", func() {
			rule := &model.Rule{Name: "rule"}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given pre, post & error tasks for this rule", func() {
				pre1 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(pre1), ShouldBeNil)
				pre2 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(pre2), ShouldBeNil)

				post1 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainPost,
					Rank:   0,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(post1), ShouldBeNil)
				post2 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainPost,
					Rank:   1,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(post2), ShouldBeNil)

				err1 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainError,
					Rank:   0,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(err1), ShouldBeNil)
				err2 := &model.Task{
					RuleID: rule.ID,
					Chain:  model.ChainError,
					Rank:   1,
					Type:   taskSuccess,
					Args:   []byte(`{}`),
				}
				So(db.Create(err2), ShouldBeNil)

				Convey("When retrieving the rule's pre-tasks", func() {
					tasks, err := GetTasks(db, rule.ID, model.ChainPre)
					So(err, ShouldBeNil)

					Convey("Then it should return the rule's pre-tasks", func() {
						So(tasks, ShouldResemble, []*model.Task{pre1, pre2})
					})
				})

				Convey("When retrieving the rule's post-tasks", func() {
					tasks, err := GetTasks(db, rule.ID, model.ChainPost)
					So(err, ShouldBeNil)

					Convey("Then it should return the rule's post-tasks", func() {
						So(tasks, ShouldResemble, []*model.Task{post1, post2})
					})
				})

				Convey("When retrieving the rule's error-tasks", func() {
					tasks, err := GetTasks(db, rule.ID, model.ChainError)
					So(err, ShouldBeNil)

					Convey("Then it should return the rule's error-tasks", func() {
						So(tasks, ShouldResemble, []*model.Task{err1, err2})
					})
				})
			})
		})
	})
}

func TestRunTasks(t *testing.T) {
	logger := log.NewLogger("test_run_tasks", logConf)

	Convey("Given a processor", t, func() {
		db := database.GetTestDatabase()
		shutdown := make(chan bool)

		rule := &model.Rule{Name: "rule"}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.RemoteAgent{
			Name:        "agent",
			Protocol:    "r66",
			ProtoConfig: []byte(`{}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: agent.ID,
			Login:         "login",
			Password:      []byte("password"),
		}
		So(db.Create(account), ShouldBeNil)

		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   false,
			AgentID:    agent.ID,
			AccountID:  account.ID,
			SourcePath: "source",
			DestPath:   "tasks_runner_test.go",
		}
		So(db.Create(trans), ShouldBeNil)

		proc := &Processor{
			Db:       db,
			Logger:   logger,
			Rule:     rule,
			Transfer: trans,
			Shutdown: shutdown,
		}

		Convey("Given a list of tasks", func() {

			Convey("Given that all the tasks succeed", func() {
				dummyTaskCheck = make(chan string, 3)
				tasks := []*model.Task{
					{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   taskSuccess,
						Args:   []byte(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskWarning,
						Args:   []byte(`{}`),
					},
				}

				Convey("Then it should run the tasks without error", func() {
					So(proc.RunTasks(tasks), ShouldBeNil)
					dummyTaskCheck <- "DONE"

					Convey("Then it should have executed all tasks", func() {
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "WARNING")
						So(<-dummyTaskCheck, ShouldEqual, "DONE")

						Convey("Then the transfer should have a warning", func() {
							So(trans.Error.Code, ShouldEqual, model.TeWarning)
							So(trans.Error.Details, ShouldEqual, "Task TESTWARNING @ rule PRE[1]: warning message")
						})
					})
				})
			})

			Convey("Given that one of the tasks fails", func() {
				dummyTaskCheck = make(chan string, 3)

				tasks := []*model.Task{
					{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   taskSuccess,
						Args:   []byte(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskFail,
						Args:   []byte(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskSuccess,
						Args:   []byte(`{}`),
					},
				}

				Convey("Then running the tasks should return an error", func() {
					So(proc.RunTasks(tasks), ShouldNotBeNil)
					dummyTaskCheck <- "DONE"

					Convey("Then it should have executed all tasks up to the failed one", func() {
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "FAILURE")
						So(<-dummyTaskCheck, ShouldEqual, "DONE")

						Convey("Then the transfer should have an error", func() {
							So(trans.Error.Code, ShouldEqual, model.TeExternalOperation)
							So(trans.Error.Details, ShouldEqual, "Task TESTFAIL @ rule PRE[1]: task failed")
						})
					})
				})
			})

			Convey("Given that one of the tasks is invalid", func() {
				dummyTaskCheck = make(chan string, 1)

				tasks := []*model.Task{{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   taskSuccess,
					Args:   []byte(`{`),
				}}

				Convey("Then running the tasks should return an error", func() {
					So(proc.RunTasks(tasks), ShouldNotBeNil)
				})
			})

			Convey("Given an unknown type of task", func() {
				dummyTaskCheck = make(chan string, 1)

				tasks := []*model.Task{{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   "unknown",
					Args:   []byte(`{}`),
				}}

				Convey("Then running the tasks should return an error", func() {
					So(proc.RunTasks(tasks), ShouldNotBeNil)
				})
			})
		})
	})
}
