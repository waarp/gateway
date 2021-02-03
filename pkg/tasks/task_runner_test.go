package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
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

		Convey("Given a Runner", func(c C) {
			db := database.TestDatabase(c, "ERROR")

			agent := &model.RemoteAgent{
				Name:        "agent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6622",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "account",
				Password:      []byte("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			r := &Runner{
				db: db,
				info: &model.TransferContext{
					Rule: &model.Rule{
						Name:    "Test",
						IsSend:  true,
						Path:    "path/to/test",
						OutPath: "out/path",
					},
					Transfer: &model.Transfer{
						ID:        1234,
						IsServer:  false,
						AgentID:   agent.ID,
						AccountID: account.ID,
						Error: types.TransferError{
							Details: `", "bad":1`,
						},
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
							So(val, ShouldEqual, r.info.Rule.Name)
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

					Convey("Then res should contain an entry `trueFullPath`", func() {
						val, ok := res["trueFullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.info.Transfer.SourceFile)
						})
					})

					Convey("Then res should contain an entry `trueFilename`", func() {
						val, ok := res["trueFilename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFilename] should contain the resolved variable", func() {
							So(val, ShouldEqual, filepath.Base(r.info.Transfer.SourceFile))
						})
					})

					Convey("Then res should contain an entry `fullPath`", func() {
						val, ok := res["fullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[fullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.info.Transfer.SourceFile)
						})
					})

					Convey("Then res should contain an entry `filename`", func() {
						val, ok := res["filename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[filename] should contain the resolved variable", func() {
							So(val, ShouldEqual, filepath.Base(r.info.Transfer.SourceFile))
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
							So(val, ShouldEqual, fmt.Sprint(r.info.Transfer.ID))
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
							So(val, ShouldEqual, fmt.Sprintf("%d_%s_%s",
								r.info.Transfer.ID, account.Login, agent.Name))
						})
					})

					Convey("Then res should contain an entry `errCode`", func() {
						val, ok := res["errCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, string(r.info.Transfer.Error.Code.R66Code()))
						})
					})

					Convey("Then res should contain an entry `errMsg`", func() {
						val, ok := res["errMsg"]
						So(ok, ShouldBeTrue)

						Convey("Then res[msg] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.info.Transfer.Error.Details)
						})
					})

					Convey("Then res should contain an entry `errStrCode`", func() {
						val, ok := res["errStrCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errStrCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.info.Transfer.Error.Details)
						})
					})
				})
			})
		})
	})
}

func TestRunTasks(t *testing.T) {
	logger := log.NewLogger("test_run_tasks")

	Convey("Given a processor", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		rule := &model.Rule{Name: "rule", IsSend: false, Path: "path"}
		So(db.Insert(rule).Run(), ShouldBeNil)

		agent := &model.RemoteAgent{
			Name:        "agent",
			Protocol:    "test",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:6622",
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: agent.ID,
			Login:         "login",
			Password:      []byte("password"),
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   false,
			AgentID:    agent.ID,
			AccountID:  account.ID,
			SourceFile: "source",
			DestFile:   "task_runner_test.go",
		}
		So(db.Insert(trans).Run(), ShouldBeNil)

		proc := &Runner{
			db:     db,
			logger: logger,
			info: &model.TransferContext{
				Rule:     rule,
				Transfer: trans,
			},
			ctx: context.Background(),
		}

		Convey("Given a list of tasks", func() {

			Convey("Given that all the tasks succeed", func() {
				dummyTaskCheck = make(chan string, 3)
				tasks := []model.Task{
					{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   taskSuccess,
						Args:   json.RawMessage(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskSuccess,
						Args:   json.RawMessage(`{}`),
					},
				}

				Convey("Then it should run the tasks without error", func() {
					rv := proc.runTasks(tasks)
					So(rv, ShouldBeNil)

					dummyTaskCheck <- "DONE"

					Convey("Then it should have executed all tasks", func() {
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "DONE")
					})
				})
			})

			Convey("Given that one task is in warning", func() {
				dummyTaskCheck = make(chan string, 3)
				tasks := []model.Task{
					{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   taskSuccess,
						Args:   json.RawMessage(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskWarning,
						Args:   json.RawMessage(`{}`),
					},
				}

				Convey("Then it should run the tasks without error", func() {
					rv := proc.runTasks(tasks)
					So(rv, ShouldBeNil)

					dummyTaskCheck <- "DONE"

					Convey("Then it should have executed all tasks", func() {
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "WARNING")
						So(<-dummyTaskCheck, ShouldEqual, "DONE")

						Convey("Then the transfer should have a warning", func() {
							So(trans.Error.Code, ShouldEqual, types.TeWarning)
							So(trans.Error.Details, ShouldEqual, "Task TESTWARNING @ rule PRE[1]: warning message")
						})
					})
				})
			})

			Convey("Given that one of the tasks fails", func() {
				dummyTaskCheck = make(chan string, 3)

				tasks := []model.Task{
					{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   taskSuccess,
						Args:   json.RawMessage(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskFail,
						Args:   json.RawMessage(`{}`),
					}, {
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   taskSuccess,
						Args:   json.RawMessage(`{}`),
					},
				}

				Convey("Then running the tasks should return an error", func() {
					rv := proc.runTasks(tasks)
					So(rv, ShouldBeError)
					dummyTaskCheck <- "DONE"

					Convey("Then it should have executed all tasks up to the failed one", func() {
						So(<-dummyTaskCheck, ShouldEqual, "SUCCESS")
						So(<-dummyTaskCheck, ShouldEqual, "FAILURE")
						So(<-dummyTaskCheck, ShouldEqual, "DONE")
					})
				})
			})

			Convey("Given that one of the tasks is invalid", func() {
				dummyTaskCheck = make(chan string, 1)

				tasks := []model.Task{{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   taskSuccess,
					Args:   []byte(`{`),
				}}

				Convey("Then running the tasks should return an error", func() {
					So(proc.runTasks(tasks), ShouldNotBeNil)
				})
			})

			Convey("Given an unknown type of task", func() {
				dummyTaskCheck = make(chan string, 1)

				tasks := []model.Task{{
					RuleID: rule.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   "unknown",
					Args:   json.RawMessage(`{}`),
				}}

				Convey("Then running the tasks should return an error", func() {
					So(proc.runTasks(tasks), ShouldNotBeNil)
				})
			})
		})
	})
}
