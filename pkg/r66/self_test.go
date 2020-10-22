package r66

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"
)

var testFileContent = []byte("r66 self transfer test file")

type testContext struct {
	logger                   *log.Logger
	db                       *database.DB
	clientPaths, serverPaths pipeline.Paths

	server     *model.LocalAgent
	locAccount *model.LocalAccount
	partner    *model.RemoteAgent
	remAccount *model.RemoteAccount
	send, recv *model.Rule

	trans *model.Transfer
}

func initForSelfTransfer(c C) *testContext {
	logger := log.NewLogger("r66_self_transfer")
	db := database.GetTestDatabase()
	home := testhelpers.TempDir(c, "r66_self_transfer")
	port := testhelpers.GetFreePort(c)

	pathConf := conf.PathsConfig{
		GatewayHome:   home,
		InDirectory:   home,
		OutDirectory:  home,
		WorkDirectory: filepath.Join(home, "tmp"),
	}
	clientPaths := pipeline.Paths{PathsConfig: pathConf}

	root := filepath.Join(home, "r66_server_root")
	So(os.MkdirAll(root, 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(root, "in"), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(root, "out"), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(root, "work"), 0o700), ShouldBeNil)

	serverPaths := pipeline.Paths{
		PathsConfig: pathConf,
		ServerRoot:  root,
		ServerIn:    "in",
		ServerOut:   "out",
		ServerWork:  "work",
	}

	server := &model.LocalAgent{
		Name:        "r66_server",
		Protocol:    "r66",
		Root:        utils.NormalizePath(root),
		ProtoConfig: json.RawMessage(`{"blockSize":50,"serverPassword":"c2VzYW1l"}`),
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	So(db.Create(server), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
		Password:     r66.CryptPass([]byte("sesame")),
	}
	So(db.Create(locAccount), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:        "r66_partner",
		Protocol:    "r66",
		ProtoConfig: json.RawMessage(`{"blockSize":50}`),
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	So(db.Create(partner), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "toto",
		Password:      []byte("sesame"),
	}
	So(db.Create(remAccount), ShouldBeNil)

	service := NewService(db, server, logger)
	So(service.Start(), ShouldBeNil)
	service.server.AuthentHandler = &testAuthHandler{service.server.AuthentHandler}
	Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		So(service.Stop(ctx), ShouldBeNil)
	})

	service.paths = pipeline.Paths{
		PathsConfig: pathConf,
		ServerRoot:  server.Root,
		ServerIn:    server.InDir,
		ServerOut:   server.OutDir,
		ServerWork:  server.WorkDir,
	}

	send := &model.Rule{
		Name:   "self",
		Path:   "/send",
		IsSend: true,
	}
	So(db.Create(send), ShouldBeNil)

	sPreTask1 := &model.Task{
		RuleID: send.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"SEND | PRE-TASKS[0] | OK"}`),
	}
	So(db.Create(sPreTask1), ShouldBeNil)

	sPostTask1 := &model.Task{
		RuleID: send.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"SEND | POST-TASKS[0] | OK"}`),
	}
	So(db.Create(sPostTask1), ShouldBeNil)

	sErrTask1 := &model.Task{
		RuleID: send.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"SEND | ERROR-TASKS[0] | OK"}`),
	}
	So(db.Create(sErrTask1), ShouldBeNil)

	recv := &model.Rule{
		Name:   "self",
		Path:   "/recv",
		IsSend: false,
	}
	So(db.Create(recv), ShouldBeNil)

	rPreTask1 := &model.Task{
		RuleID: recv.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"RECV | PRE-TASKS[0] | OK"}`),
	}
	So(db.Create(rPreTask1), ShouldBeNil)

	rPostTask1 := &model.Task{
		RuleID: recv.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"RECV | POST-TASKS[0] | OK"}`),
	}
	So(db.Create(rPostTask1), ShouldBeNil)

	rErrTask1 := &model.Task{
		RuleID: recv.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   "TESTCHECK",
		Args:   json.RawMessage(`{"msg":"RECV | ERROR-TASKS[0] | OK"}`),
	}
	So(db.Create(rErrTask1), ShouldBeNil)

	return &testContext{
		logger:      logger,
		db:          db,
		clientPaths: clientPaths,
		serverPaths: serverPaths,
		server:      server,
		locAccount:  locAccount,
		partner:     partner,
		remAccount:  remAccount,
		send:        send,
		recv:        recv,
	}
}

func initTransfer(ctx *testContext, isPush bool) *model.Transfer {
	if isPush {
		testFile := filepath.Join(ctx.clientPaths.GatewayHome, "r66_self_transfer_push.src")
		So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

		trans := &model.Transfer{
			RuleID:       ctx.send.ID,
			IsServer:     false,
			AgentID:      ctx.server.ID,
			AccountID:    ctx.locAccount.ID,
			TrueFilepath: testFile,
			SourceFile:   "r66_self_transfer_push.src",
			DestFile:     "r66_self_transfer_push.dst",
			Start:        time.Now(),
		}
		So(ctx.db.Create(trans), ShouldBeNil)

		return trans
	}

	testFile := filepath.Join(ctx.serverPaths.ServerRoot,
		ctx.serverPaths.ServerOut, "r66_self_transfer_pull.src")
	So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

	trans := &model.Transfer{
		RuleID:       ctx.recv.ID,
		IsServer:     false,
		AgentID:      ctx.server.ID,
		AccountID:    ctx.locAccount.ID,
		TrueFilepath: testFile,
		SourceFile:   "r66_self_transfer_pull.src",
		DestFile:     "r66_self_transfer_pull.dst",
		Start:        time.Now(),
	}
	So(ctx.db.Create(trans), ShouldBeNil)

	return trans
}

func processTransfer(ctx *testContext) {
	stream, err := pipeline.NewTransferStream(context.Background(),
		ctx.logger, ctx.db, ctx.clientPaths, ctx.trans)
	So(err, ShouldBeNil)

	exe := executor.Executor{TransferStream: stream}
	checkChannel = make(chan string, 10)
	exe.Run()
	checkChannel <- "CLIENT END TRANSFER"
}

func checkHistory(ctx *testContext) {
	Convey("Then the transfers should be over", func() {
		var transfers []model.Transfer
		So(ctx.db.Select(&transfers, nil), ShouldBeNil)
		So(transfers, ShouldBeEmpty)

		var results []model.TransferHistory
		So(ctx.db.Select(&results, nil), ShouldBeNil)
		So(len(results), ShouldEqual, 2)

		Convey("Then there should be a client-side history entry", func() {
			cTrans := model.TransferHistory{
				ID:             ctx.trans.ID,
				Owner:          database.Owner,
				Protocol:       "r66",
				IsServer:       false,
				Account:        ctx.remAccount.Login,
				Agent:          ctx.partner.Name,
				Start:          results[0].Start,
				Stop:           results[0].Stop,
				SourceFilename: ctx.trans.SourceFile,
				DestFilename:   ctx.trans.DestFile,
				Status:         model.StatusDone,
				Step:           model.StepNone,
				Error:          model.TransferError{},
				Progress:       uint64(len(testFileContent)),
				TaskNumber:     0,
			}
			if ctx.trans.RuleID == ctx.send.ID {
				cTrans.IsSend = true
				cTrans.Rule = ctx.send.Name
			} else {
				cTrans.IsSend = false
				cTrans.Rule = ctx.recv.Name
			}
			So(results[0], ShouldResemble, cTrans)
		})

		Convey("Then there should be a server-side history entry", func() {
			sTrans := model.TransferHistory{
				ID:               ctx.trans.ID + 1,
				RemoteTransferID: fmt.Sprint(ctx.trans.ID),
				Owner:            database.Owner,
				Protocol:         "r66",
				IsServer:         true,
				Account:          ctx.locAccount.Login,
				Agent:            ctx.server.Name,
				Start:            results[1].Start,
				Stop:             results[1].Stop,
				Status:           model.StatusDone,
				Step:             model.StepNone,
				Error:            model.TransferError{},
				Progress:         uint64(len(testFileContent)),
				TaskNumber:       0,
			}
			if ctx.trans.RuleID == ctx.send.ID {
				sTrans.IsSend = false
				sTrans.Rule = ctx.recv.Name
				sTrans.SourceFilename = ctx.trans.DestFile
				sTrans.DestFilename = ctx.trans.DestFile
			} else {
				sTrans.IsSend = true
				sTrans.Rule = ctx.send.Name
				sTrans.SourceFilename = ctx.trans.SourceFile
				sTrans.DestFilename = ctx.trans.SourceFile
			}
			So(results[1], ShouldResemble, sTrans)
		})
	})
}

func checkTransfers(ctx *testContext, cTrans, sTrans *model.Transfer) {
	Convey("Then the transfers should be over", func() {
		var results []model.Transfer
		So(ctx.db.Select(&results, nil), ShouldBeNil)
		So(len(results), ShouldEqual, 2)

		Convey("Then there should be a client-side transfer entry in error", func() {
			cTrans.Owner = database.Owner
			cTrans.ID = ctx.trans.ID
			cTrans.IsServer = false
			cTrans.RuleID = ctx.trans.RuleID
			cTrans.AccountID = ctx.remAccount.ID
			cTrans.AgentID = ctx.partner.ID
			cTrans.Start = results[0].Start
			cTrans.SourceFile = ctx.trans.SourceFile
			cTrans.DestFile = ctx.trans.DestFile
			cTrans.TrueFilepath = ctx.trans.TrueFilepath
			So(results[0], ShouldResemble, *cTrans)
		})

		Convey("Then there should be a server-side transfer entry in error", func() {
			sTrans.Owner = database.Owner
			sTrans.ID = ctx.trans.ID + 1
			sTrans.RemoteTransferID = fmt.Sprint(ctx.trans.ID)
			sTrans.IsServer = true
			sTrans.AccountID = ctx.locAccount.ID
			sTrans.AgentID = ctx.server.ID
			sTrans.Start = results[1].Start
			if ctx.trans.RuleID == ctx.send.ID {
				sTrans.RuleID = ctx.recv.ID
				sTrans.SourceFile = ctx.trans.DestFile
				sTrans.DestFile = ctx.trans.DestFile
				if sTrans.Step > model.StepData {
					sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
						ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerIn,
						ctx.trans.DestFile))
				} else {
					sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
						ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerWork,
						ctx.trans.DestFile+".tmp"))
				}
			} else {
				sTrans.RuleID = ctx.send.ID
				sTrans.SourceFile = ctx.trans.SourceFile
				sTrans.DestFile = ctx.trans.SourceFile
				sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
					ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerOut,
					ctx.trans.SourceFile))
			}
			So(results[1], ShouldResemble, *sTrans)
		})
	})
}

func checkFile(ctx *testContext) {
	Convey("Then the file should have been sent entirely", func() {
		path := filepath.Join(ctx.clientPaths.GatewayHome, ctx.trans.DestFile)
		if ctx.trans.RuleID == ctx.send.ID {
			path = filepath.Join(ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerIn,
				ctx.trans.DestFile)
		}
		content, err := ioutil.ReadFile(path)
		So(err, ShouldBeNil)
		So(string(content), ShouldEqual, string(testFileContent))
	})
}

func TestSelfPush(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func() {
			ctx.trans = initTransfer(ctx, true)

			Convey("Once the transfer has been processed", func() {
				processTransfer(ctx)

				Convey("Then it should have executed all the tasks in order", func() {
					So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
					shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
					close(checkChannel)

					checkHistory(ctx)
					checkFile(ctx)
				})
			})
		})
	})
}

func TestSelfPull(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func() {
			ctx.trans = initTransfer(ctx, false)

			Convey("Once the transfer has been processed", func() {
				processTransfer(ctx)

				Convey("Then it should have executed all the tasks in order", func() {
					So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
					shouldFinishOK("CLIENT END TRANSFER", "SERVER END TRANSFER OK")
					close(checkChannel)

					checkHistory(ctx)
					checkFile(ctx)
				})
			})
		})
	})
}

func TestSelfPushClientPreTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func() {
			ctx.trans = initTransfer(ctx, true)

			Convey("Given an error during the client's pre-tasks", func() {
				sPreTask2 := &model.Task{
					RuleID: ctx.send.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"SEND | PRE-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(sPreTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[1] | FAIL")
						shouldFinishError(
							"SEND | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"RECV | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPreTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   0,
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPreTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   0,
							TaskNumber: 1,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(sPreTask2), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									checkFile(ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushServerPreTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func() {
			ctx.trans = initTransfer(ctx, true)

			rPreTask2 := &model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   "TESTFAIL",
				Args:   json.RawMessage(`{"msg":"RECV | PRE-TASKS[1] | FAIL"}`),
			}
			So(ctx.db.Create(rPreTask2), ShouldBeNil)

			Convey("Once the transfer has been processed", func() {
				processTransfer(ctx)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
					So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[1] | FAIL")
					shouldFinishError(
						"SEND | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
						"RECV | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
					close(checkChannel)

					cTrans := &model.Transfer{
						Status: model.StatusError,
						Step:   model.StepSetup,
						Error: model.TransferError{
							Code:    model.TeExternalOperation,
							Details: "Task TESTFAIL @ self PRE[1]: task failed",
						},
						Progress:   0,
						TaskNumber: 0,
					}

					sTrans := &model.Transfer{
						Status: model.StatusError,
						Step:   model.StepPreTasks,
						Error: model.TransferError{
							Code:    model.TeExternalOperation,
							Details: "Task TESTFAIL @ self PRE[1]: task failed",
						},
						Progress:   0,
						TaskNumber: 1,
					}

					checkTransfers(ctx, cTrans, sTrans)

					Convey("When retrying the transfer", func() {
						So(ctx.db.Delete(rPreTask2), ShouldBeNil)

						retry := &model.Transfer{ID: ctx.trans.ID}
						So(ctx.db.Get(retry), ShouldBeNil)
						retry.Status = model.StatusPlanned
						So(ctx.db.Update(retry), ShouldBeNil)
						ctx.trans = retry

						Convey("Once the transfer has been processed", func() {
							processTransfer(ctx)

							Convey("Then it should have executed all the tasks in order", func(c C) {
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
								So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
								shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
								close(checkChannel)

								checkHistory(ctx)
								checkFile(ctx)
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullClientPreTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func() {
			ctx.trans = initTransfer(ctx, false)

			Convey("Given an error during the client's pre-tasks", func() {
				rPreTask2 := &model.Task{
					RuleID: ctx.recv.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"RECV | PRE-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(rPreTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[1] | FAIL")
						shouldFinishError(
							"RECV | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"SEND | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPreTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   0,
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepData,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(rPreTask2), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									checkFile(ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullServerPreTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func() {
			ctx.trans = initTransfer(ctx, false)

			Convey("Given an error during the server's pre-tasks", func() {
				sPreTask2 := &model.Task{
					RuleID: ctx.send.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"SEND | PRE-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(sPreTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[1] | FAIL")
						shouldFinishError(
							"RECV | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"SEND | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepSetup,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   0,
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPreTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self PRE[1]: task failed",
							},
							Progress:   0,
							TaskNumber: 1,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(sPreTask2), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
									So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									checkFile(ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushClientPostTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func() {
			ctx.trans = initTransfer(ctx, true)

			Convey("Given an error during the client's post-tasks", func() {
				sPostTask2 := &model.Task{
					RuleID: ctx.send.ID,
					Chain:  model.ChainPost,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"SEND | POST-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(sPostTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[1] | FAIL")
						shouldFinishError(
							"SEND | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"RECV | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPostTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPostTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(sPostTask2), ShouldBeNil)
							sPostTask2b := &model.Task{
								RuleID: ctx.send.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTCHECK",
								Args:   json.RawMessage(`{"msg":"SEND | POST-TASKS[1] | OK"}`),
							}
							So(ctx.db.Create(sPostTask2b), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[1] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									//checkFile(ctx)
									// file cannot be checked as it is deleted between tests
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushServerPostTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func() {
			ctx.trans = initTransfer(ctx, true)

			Convey("Given an error during the server's post-tasks", func() {
				rPostTask2 := &model.Task{
					RuleID: ctx.recv.ID,
					Chain:  model.ChainPost,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"RECV | POST-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(rPostTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[1] | FAIL")
						shouldFinishError(
							"SEND | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"RECV | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepData,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPostTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(rPostTask2), ShouldBeNil)
							rPostTask2b := &model.Task{
								RuleID: ctx.recv.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTCHECK",
								Args:   json.RawMessage(`{"msg":"RECV | POST-TASKS[1] | OK"}`),
							}
							So(ctx.db.Create(rPostTask2b), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[1] | OK")
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									//checkFile(ctx)
									// file cannot be checked as it is deleted between tests
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullClientPostTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func() {
			ctx.trans = initTransfer(ctx, false)

			Convey("Given an error during the client's post-tasks", func() {
				rPostTask2 := &model.Task{
					RuleID: ctx.recv.ID,
					Chain:  model.ChainPost,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"RECV | POST-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(rPostTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[1] | FAIL")
						shouldFinishError(
							"RECV | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"SEND | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPostTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepData,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(rPostTask2), ShouldBeNil)
							rPostTask2b := &model.Task{
								RuleID: ctx.recv.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTCHECK",
								Args:   json.RawMessage(`{"msg":"RECV | POST-TASKS[1] | OK"}`),
							}
							So(ctx.db.Create(rPostTask2b), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[1] | OK")
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									//checkFile(ctx)
									// file cannot be checked as it is deleted between tests
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullServerPostTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func() {
			ctx.trans = initTransfer(ctx, false)

			Convey("Given an error during the server's post-tasks", func() {
				sPostTask2 := &model.Task{
					RuleID: ctx.send.ID,
					Chain:  model.ChainPost,
					Rank:   1,
					Type:   "TESTFAIL",
					Args:   json.RawMessage(`{"msg":"SEND | POST-TASKS[1] | FAIL"}`),
				}
				So(ctx.db.Create(sPostTask2), ShouldBeNil)

				Convey("Once the transfer has been processed", func() {
					processTransfer(ctx)

					Convey("Then it should have executed all the tasks in order", func() {
						So(<-checkChannel, ShouldEqual, "SEND | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | PRE-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "RECV | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[0] | OK")
						So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[1] | FAIL")
						shouldFinishError(
							"RECV | ERROR-TASKS[0] | OK", "CLIENT END TRANSFER",
							"SEND | ERROR-TASKS[0] | OK", "SERVER END TRANSFER ERROR")
						close(checkChannel)

						cTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepFinalization,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Status: model.StatusError,
							Step:   model.StepPostTasks,
							Error: model.TransferError{
								Code:    model.TeExternalOperation,
								Details: "Task TESTFAIL @ self POST[1]: task failed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfers(ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func() {
							So(ctx.db.Delete(sPostTask2), ShouldBeNil)
							sPostTask2b := &model.Task{
								RuleID: ctx.send.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTCHECK",
								Args:   json.RawMessage(`{"msg":"SEND | POST-TASKS[1] | OK"}`),
							}
							So(ctx.db.Create(sPostTask2b), ShouldBeNil)

							retry := &model.Transfer{ID: ctx.trans.ID}
							So(ctx.db.Get(retry), ShouldBeNil)
							retry.Status = model.StatusPlanned
							So(ctx.db.Update(retry), ShouldBeNil)
							ctx.trans = retry

							Convey("Once the transfer has been processed", func() {
								processTransfer(ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									So(<-checkChannel, ShouldEqual, "SEND | POST-TASKS[1] | OK")
									shouldFinishOK("SERVER END TRANSFER OK", "CLIENT END TRANSFER")
									close(checkChannel)

									checkHistory(ctx)
									checkFile(ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}
