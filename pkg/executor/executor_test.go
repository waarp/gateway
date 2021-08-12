package executor

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

func TestExecutorRun(t *testing.T) {
	logger := log.NewLogger("test_executor_run")

	Convey("Given a database", t, func(c C) {
		root := testhelpers.TempDir(c, "test_executor_run")
		paths := pipeline.Paths{PathsConfig: conf.PathsConfig{
			GatewayHome:   root,
			InDirectory:   root,
			OutDirectory:  root,
			WorkDirectory: filepath.Join(root, "work"),
		}}

		db := database.TestDatabase(c, "ERROR")

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
		}
		So(db.Insert(remote).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      "test password",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		Convey("Given an outgoing transfer", func() {
			rule := &model.Rule{
				Name:   "test_rule",
				IsSend: true,
				Path:   ".",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			content := []byte("executor test run file content")
			truePath := filepath.Join(paths.OutDirectory, "test_run_src")
			So(ioutil.WriteFile(truePath, content, 0o700), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:       rule.ID,
				AgentID:      remote.ID,
				AccountID:    account.ID,
				TrueFilepath: truePath,
				SourceFile:   filepath.Base(truePath),
				DestFile:     "test_run_dst",
				Start:        time.Now().Round(time.Microsecond),
				Status:       types.StatusPlanned,
				Owner:        database.Owner,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given an executor", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{TransferStream: stream}

				Convey("Given that the transfer is successful", func() {
					ClientsConstructors["test"] = NewAllSuccess

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							var res model.Transfers
							So(db.Select(&res).Run(), ShouldBeNil)
							So(res, ShouldBeEmpty)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist", func() {
							var results model.Histories
							So(db.Select(&results).Run(), ShouldBeNil)
							So(results, ShouldNotBeEmpty)

							expected := model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Agent:          remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourceFile,
								DestFilename:   trans.DestFile,
								Rule:           rule.Name,
								Start:          trans.Start,
								Stop:           results[0].Stop,
								Status:         types.StatusDone,
								Step:           types.StepNone,
								Progress:       uint64(len(content)),
								TaskNumber:     0,
							}

							So(results[0], ShouldResemble, expected)
						})
					})
				})

				Convey("Given that the connection fails", func() {
					ClientsConstructors["test"] = NewConnectFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepSetup,
								Error: types.TransferError{
									Code:    types.TeConnection,
									Details: "connection failed",
								},
								Progress:   0,
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the authentication fails", func() {
					ClientsConstructors["test"] = NewAuthFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepSetup,
								Error: types.TransferError{
									Code:    types.TeBadAuthentication,
									Details: "authentication failed",
								},
								Progress:   0,
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the request fails", func() {
					ClientsConstructors["test"] = NewRequestFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepSetup,
								Error: types.TransferError{
									Code:    types.TeForbidden,
									Details: "request failed",
								},
								Progress:   0,
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the pre-tasks fail", func() {
					ClientsConstructors["test"] = NewAllSuccess

					preTask := &model.Task{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Insert(preTask).Run(), ShouldBeNil)

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepPreTasks,
								Error: types.TransferError{
									Code:    types.TeExternalOperation,
									Details: "Task TESTFAIL @ test_rule PRE[0]: task failed",
								},
								Progress:   0,
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the data transfer fails", func() {
					ClientsConstructors["test"] = NewDataFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepData,
								Error: types.TransferError{
									Code:    types.TeDataTransfer,
									Details: "data failed",
								},
								Progress:   0,
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the post-tasks fail", func() {
					ClientsConstructors["test"] = NewAllSuccess

					preTask := &model.Task{
						RuleID: rule.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Insert(preTask).Run(), ShouldBeNil)

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepPostTasks,
								Error: types.TransferError{
									Code:    types.TeExternalOperation,
									Details: "Task TESTFAIL @ test_rule POST[0]: task failed",
								},
								Progress:   uint64(len(content)),
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})

				Convey("Given that the remote post-tasks fail", func() {
					ClientsConstructors["test"] = NewCloseFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should be in error", func() {
							exp := model.Transfer{
								ID:           trans.ID,
								Owner:        trans.Owner,
								IsServer:     false,
								AccountID:    account.ID,
								AgentID:      remote.ID,
								TrueFilepath: trans.TrueFilepath,
								SourceFile:   trans.SourceFile,
								DestFile:     trans.DestFile,
								RuleID:       rule.ID,
								Start:        trans.Start,
								Status:       types.StatusError,
								Step:         types.StepFinalization,
								Error: types.TransferError{
									Code:    types.TeExternalOperation,
									Details: "remote post-tasks failed",
								},
								Progress:   uint64(len(content)),
								TaskNumber: 0,
							}

							var t model.Transfers
							So(db.Select(&t).Run(), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, exp)
						})
					})
				})
			})
		})
	})
}

func TestTransferResume(t *testing.T) {
	logger := log.NewLogger("test_transfer_resume")

	Convey("Given a test database", t, func(c C) {
		root := testhelpers.TempDir(c, "test_transfer_resume_root")
		paths := pipeline.Paths{PathsConfig: conf.PathsConfig{
			GatewayHome:   root,
			InDirectory:   root,
			OutDirectory:  root,
			WorkDirectory: filepath.Join(root, "work"),
		}}

		db := database.TestDatabase(c, "ERROR")

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
		}
		So(db.Insert(remote).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      "test password",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		rule := &model.Rule{
			Name:   "resume",
			IsSend: true,
			Path:   ".",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given a transfer interrupted during pre-tasks", func() {
			ClientsConstructors["test"] = NewAllSuccess

			pre := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "TESTSUCCESS",
				Args:   []byte("{}"),
			}
			So(db.Insert(pre).Run(), ShouldBeNil)

			content := []byte("test pre-tasks file content")
			truePath := filepath.Join(paths.OutDirectory, "test_pre_tasks_src")
			So(ioutil.WriteFile(truePath, content, 0o700), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:       rule.ID,
				IsServer:     false,
				AgentID:      remote.ID,
				AccountID:    account.ID,
				TrueFilepath: truePath,
				SourceFile:   "test_pre_tasks_src",
				DestFile:     "test_pre_tasks_dst",
				Start:        time.Now().Round(time.Microsecond),
				Step:         types.StepPreTasks,
				Status:       types.StatusPlanned,
				Owner:        database.Owner,
				Progress:     0,
				TaskNumber:   1,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{
					TransferStream: stream,
				}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					var res model.Transfers
					So(db.Select(&res).Run(), ShouldBeNil)
					So(res, ShouldBeEmpty)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h model.Histories
					So(db.Select(&h).Run(), ShouldBeNil)
					So(h, ShouldNotBeEmpty)

					hist := model.TransferHistory{
						ID:             trans.ID,
						Owner:          trans.Owner,
						IsServer:       false,
						IsSend:         true,
						Account:        account.Login,
						Agent:          remote.Name,
						Protocol:       remote.Protocol,
						SourceFilename: trans.SourceFile,
						DestFilename:   trans.DestFile,
						Rule:           rule.Name,
						Start:          trans.Start,
						Stop:           h[0].Stop,
						Step:           types.StepNone,
						Status:         types.StatusDone,
						Progress:       uint64(len(content)),
					}

					So(h[0], ShouldResemble, hist)
				})
			})
		})

		Convey("Given a transfer interrupted during data transfer", func() {
			ClientsConstructors["test"] = NewAllSuccess

			pre := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "TESTFAIL",
				Args:   []byte("{}"),
			}
			So(db.Insert(pre).Run(), ShouldBeNil)

			content := []byte("test data file content")
			truePath := filepath.Join(paths.OutDirectory, "test_data_src")
			So(ioutil.WriteFile(truePath, content, 0o700), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:       rule.ID,
				IsServer:     false,
				AgentID:      remote.ID,
				AccountID:    account.ID,
				TrueFilepath: truePath,
				SourceFile:   "test_data_src",
				DestFile:     "test_data_dst",
				Start:        time.Now().Round(time.Microsecond),
				Step:         types.StepData,
				Status:       types.StatusPlanned,
				Owner:        database.Owner,
				Progress:     10,
				TaskNumber:   0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{TransferStream: stream}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					var res model.Transfers
					So(db.Select(&res).Run(), ShouldBeNil)
					So(res, ShouldBeEmpty)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h model.Histories
					So(db.Select(&h).Run(), ShouldBeNil)
					So(h, ShouldNotBeEmpty)

					hist := model.TransferHistory{
						ID:             trans.ID,
						Owner:          trans.Owner,
						IsServer:       false,
						IsSend:         true,
						Account:        account.Login,
						Agent:          remote.Name,
						Protocol:       remote.Protocol,
						SourceFilename: trans.SourceFile,
						DestFilename:   trans.DestFile,
						Rule:           rule.Name,
						Start:          trans.Start,
						Stop:           h[0].Stop,
						Step:           types.StepNone,
						Status:         types.StatusDone,
						Progress:       uint64(len(content)),
					}

					So(h[0], ShouldResemble, hist)
				})
			})
		})

		Convey("Given a transfer interrupted during post tasks", func() {
			ClientsConstructors["test"] = NewDataFail

			pre := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "TESTFAIL",
				Args:   []byte("{}"),
			}
			post := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "TESTSUCCESS",
				Args:   []byte("{}"),
			}
			So(db.Insert(pre).Run(), ShouldBeNil)
			So(db.Insert(post).Run(), ShouldBeNil)

			content := []byte("test post-tasks file content")
			truePath := filepath.Join(paths.OutDirectory, "test_post_tasks_src")
			So(ioutil.WriteFile(truePath, content, 0o700), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:       rule.ID,
				IsServer:     false,
				AgentID:      remote.ID,
				AccountID:    account.ID,
				TrueFilepath: truePath,
				SourceFile:   filepath.Base(truePath),
				DestFile:     "test_post_tasks_dst",
				Start:        time.Now().Round(time.Microsecond),
				Step:         types.StepPostTasks,
				Status:       types.StatusPlanned,
				Owner:        database.Owner,
				Progress:     uint64(len(content)),
				TaskNumber:   1,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{TransferStream: stream}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					var res model.Transfers
					So(db.Select(&res).Run(), ShouldBeNil)
					So(res, ShouldBeEmpty)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h model.Histories
					So(db.Select(&h).Run(), ShouldBeNil)
					So(h, ShouldNotBeEmpty)

					hist := model.TransferHistory{
						ID:             trans.ID,
						Owner:          trans.Owner,
						IsServer:       false,
						IsSend:         true,
						Account:        account.Login,
						Agent:          remote.Name,
						Protocol:       remote.Protocol,
						SourceFilename: trans.SourceFile,
						DestFilename:   trans.DestFile,
						Rule:           rule.Name,
						Start:          trans.Start,
						Stop:           h[0].Stop,
						Status:         types.StatusDone,
						Step:           types.StepNone,
						Progress:       uint64(len(content)),
					}

					So(h[0], ShouldResemble, hist)
				})
			})
		})
	})
}
