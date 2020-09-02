package executor

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
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

		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: []byte(`{}`),
		}
		So(db.Create(remote), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      []byte("test password"),
		}
		So(db.Create(account), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   remote.TableName(),
			OwnerID:     remote.ID,
			Name:        "test cert",
			PrivateKey:  nil,
			PublicKey:   []byte("public key"),
			Certificate: []byte("certificate"),
		}
		So(db.Create(cert), ShouldBeNil)

		Convey("Given an outgoing transfer", func() {
			rule := &model.Rule{
				Name:   "test_rule",
				IsSend: true,
				Path:   ".",
			}
			So(db.Create(rule), ShouldBeNil)

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
				Start:        time.Now().Truncate(time.Second),
				Status:       model.StatusPlanned,
				Owner:        database.Owner,
			}
			So(db.Create(trans), ShouldBeNil)

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
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry should exist", func() {
							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
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
								Status:         model.StatusDone,
								Step:           model.StepNone,
								Progress:       uint64(len(content)),
							}

							So(results[0], ShouldResemble, expected)
						})
					})
				})

				Convey("Given that the connection fails", func() {
					ClientsConstructors["test"] = NewConnectFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeConnection,
								"connection failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the authentication fails", func() {
					ClientsConstructors["test"] = NewAuthFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeBadAuthentication,
								"authentication failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the request fails", func() {
					ClientsConstructors["test"] = NewRequestFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeForbidden,
								"request failed")
							So(hist.Error, ShouldResemble, expErr)
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
					So(db.Create(preTask), ShouldBeNil)

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							var h []model.TransferHistory
							So(db.Select(&h, nil), ShouldBeNil)
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
								Status:         model.StatusError,
								Step:           model.StepPreTasks,
								Error: model.NewTransferError(model.TeExternalOperation,
									"Task TESTFAIL @ test_rule PRE[0]: task failed"),
							}

							So(h[0], ShouldResemble, hist)
						})
					})
				})

				Convey("Given that the data transfer fails", func() {
					ClientsConstructors["test"] = NewDataFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeDataTransfer,
								"data failed")
							So(hist.Error, ShouldResemble, expErr)
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
					So(db.Create(preTask), ShouldBeNil)

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeExternalOperation,
								"Task TESTFAIL @ test_rule POST[0]: task failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the remote post-tasks fail", func() {
					ClientsConstructors["test"] = NewCloseFail

					Convey("When calling the `Run` method", func() {
						exe.Run()

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
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
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeExternalOperation,
								"remote post-tasks failed")
							So(hist.Error, ShouldResemble, expErr)
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

		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: []byte(`{}`),
		}
		So(db.Create(remote), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      []byte("test password"),
		}
		So(db.Create(account), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   remote.TableName(),
			OwnerID:     remote.ID,
			Name:        "test cert",
			PrivateKey:  nil,
			PublicKey:   []byte("public key"),
			Certificate: []byte("certificate"),
		}
		So(db.Create(cert), ShouldBeNil)

		rule := &model.Rule{
			Name:   "resume",
			IsSend: true,
			Path:   ".",
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given a transfer interrupted during pre-tasks", func() {
			ClientsConstructors["test"] = NewAllSuccess

			pre := &model.Task{
				RuleID: rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "TESTSUCCESS",
				Args:   []byte("{}"),
			}
			So(db.Create(pre), ShouldBeNil)

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
				Start:        time.Now().Truncate(time.Second),
				Step:         model.StepPreTasks,
				Status:       model.StatusPlanned,
				Owner:        database.Owner,
				Progress:     0,
				TaskNumber:   1,
			}
			So(db.Create(trans), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{
					TransferStream: stream,
				}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					exist, err := db.Exists(trans)
					So(err, ShouldBeNil)
					So(exist, ShouldBeFalse)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h []model.TransferHistory
					So(db.Select(&h, nil), ShouldBeNil)
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
						Step:           model.StepNone,
						Status:         model.StatusDone,
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
			So(db.Create(pre), ShouldBeNil)

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
				Start:        time.Now().Truncate(time.Second),
				Step:         model.StepData,
				Status:       model.StatusPlanned,
				Owner:        database.Owner,
				Progress:     10,
				TaskNumber:   0,
			}
			So(db.Create(trans), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{TransferStream: stream}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					exist, err := db.Exists(trans)
					So(err, ShouldBeNil)
					So(exist, ShouldBeFalse)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h []model.TransferHistory
					So(db.Select(&h, nil), ShouldBeNil)
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
						Step:           model.StepNone,
						Status:         model.StatusDone,
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
			So(db.Create(pre), ShouldBeNil)
			So(db.Create(post), ShouldBeNil)

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
				Start:        time.Now().Truncate(time.Second),
				Step:         model.StepPostTasks,
				Status:       model.StatusPlanned,
				Owner:        database.Owner,
				Progress:     uint64(len(content)),
				TaskNumber:   1,
			}
			So(db.Create(trans), ShouldBeNil)

			Convey("When starting the transfer", func() {
				stream, err := pipeline.NewTransferStream(context.Background(),
					logger, db, paths, trans)
				So(err, ShouldBeNil)
				exe := &Executor{TransferStream: stream}

				exe.Run()

				Convey("Then the `Transfer` entry should no longer exist", func() {
					exist, err := db.Exists(trans)
					So(err, ShouldBeNil)
					So(exist, ShouldBeFalse)
				})

				Convey("Then the corresponding `TransferHistory` entry should exist", func() {
					var h []model.TransferHistory
					So(db.Select(&h, nil), ShouldBeNil)
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
						Status:         model.StatusDone,
						Step:           model.StepNone,
						Progress:       uint64(len(content)),
					}

					So(h[0], ShouldResemble, hist)
				})
			})
		})
	})
}

func TestBuildR66CommandArgs(t *testing.T) {
	Convey("Testing BuildR66Command", t, func() {
		Convey("Given a transfer", func() {
			oti := &model.OutTransferInfo{
				Account:  &model.RemoteAccount{Login: "foo"},
				Agent:    &model.RemoteAgent{Name: "bar"},
				Transfer: &model.Transfer{SourceFile: "path/to/my/file.txt"},
				Rule:     &model.Rule{Name: "my-super-duper-rule"},
			}

			Convey("Then it builds the correct arguments", func() {
				expected := []string{
					"foo",
					"send",
					"-to", "bar",
					"-file", "path/to/my/file.txt",
					"-rule", "my-super-duper-rule",
				}
				So(buildR66CommandArgs(oti), ShouldResemble, expected)
			})
		})
	})
}
