package sftp

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSFTPPackage(t *testing.T) {
	logger := log.NewLogger("test_sftp_package", testLogConf)

	Convey("Given an SFTP server", t, func() {
		root, err := ioutil.TempDir("", "gateway-test")
		So(err, ShouldBeNil)
		defer func() { _ = os.RemoveAll(root) }()

		listener, err := net.Listen("tcp", "localhost:0")
		So(err, ShouldBeNil)
		_, port, err := net.SplitHostPort(listener.Addr().String())
		So(err, ShouldBeNil)

		db := database.GetTestDatabase()
		localAgent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":` + port +
				`,"root":"` + root + `"}`),
		}
		So(db.Create(localAgent), ShouldBeNil)

		pwd := "tata"
		localAccount := &model.LocalAccount{
			LocalAgentID: localAgent.ID,
			Login:        "toto",
			Password:     []byte(pwd),
		}
		So(db.Create(localAccount), ShouldBeNil)

		localCert := &model.Cert{
			OwnerType:   localAgent.TableName(),
			OwnerID:     localAgent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(localCert), ShouldBeNil)

		remoteAgent := &model.RemoteAgent{
			Name:     "test_sftp_partner",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":` + port +
				`,"root":"` + root + `"}`),
		}
		So(db.Create(remoteAgent), ShouldBeNil)

		remoteAccount := &model.RemoteAccount{
			RemoteAgentID: remoteAgent.ID,
			Login:         "toto",
			Password:      []byte(pwd),
		}
		So(db.Create(remoteAccount), ShouldBeNil)

		remoteCert := &model.Cert{
			OwnerType:   remoteAgent.TableName(),
			OwnerID:     remoteAgent.ID,
			Name:        "test_sftp_partner_cert",
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(remoteCert), ShouldBeNil)

		receive := &model.Rule{
			Name:    "receive",
			Comment: "",
			IsSend:  false,
			Path:    "/path",
		}
		So(db.Create(receive), ShouldBeNil)
		send := &model.Rule{
			Name:    "send",
			Comment: "",
			IsSend:  true,
			Path:    "/path",
		}
		So(db.Create(send), ShouldBeNil)

		receivePreTask := &model.Task{
			RuleID: receive.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 1 | PRE-TASK[0]"}`),
		}
		So(db.Create(receivePreTask), ShouldBeNil)
		sendPreTask := &model.Task{
			RuleID: send.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 2 | PRE-TASK[0]"}`),
		}
		So(db.Create(sendPreTask), ShouldBeNil)

		sendPostTask := &model.Task{
			RuleID: send.ID,
			Chain:  model.ChainPost,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 2 | POST-TASK[0]"}`),
		}
		receivePostTask := &model.Task{
			RuleID: receive.ID,
			Chain:  model.ChainPost,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 1 | POST-TASK[0]"}`),
		}
		So(db.Create(sendPostTask), ShouldBeNil)
		So(db.Create(receivePostTask), ShouldBeNil)

		sendErrorTask := &model.Task{
			RuleID: send.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 2 | ERROR-TASK[0]"}`),
		}
		receiveErrorTask := &model.Task{
			RuleID: receive.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   "TESTCHECK",
			Args:   []byte(`{"msg":"TESTCHECK | Rule 1 | ERROR-TASK[0]"}`),
		}
		So(db.Create(sendErrorTask), ShouldBeNil)
		So(db.Create(receiveErrorTask), ShouldBeNil)

		config, err := loadSSHConfig(db, localCert)
		So(err, ShouldBeNil)

		ctx, cancel := context.WithCancel(context.Background())

		sshList := &sshListener{
			Db:       db,
			Logger:   logger,
			Agent:    localAgent,
			Conf:     config,
			Listener: listener,
			connWg:   sync.WaitGroup{},
			ctx:      ctx,
			cancel:   cancel,
		}
		sshList.listen()
		Reset(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = sshList.close(ctx)
		})

		Convey("Given a transfer from SFTP client to server", func() {
			start := time.Now().Truncate(time.Second)
			trans := model.Transfer{
				RuleID:     send.ID,
				IsServer:   false,
				AgentID:    remoteAgent.ID,
				AccountID:  remoteAccount.ID,
				SourcePath: "utils.go",
				DestPath:   "utils.dst",
				Start:      start,
				Status:     model.StatusPlanned,
			}
			So(db.Create(&trans), ShouldBeNil)

			Convey("Given an executor", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				stream, err := pipeline.NewTransferStream(ctx, logger, db, "", trans)
				So(err, ShouldBeNil)

				exe := executor.Executor{
					TransferStream: stream,
					Ctx:            ctx,
				}
				finished := make(chan bool)
				go func() {
					exe.Run()
					close(finished)
				}()

				Convey("When launching the transfer with the client", func() {
					So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | PRE-TASK[0]")
					So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | PRE-TASK[0]")
					So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | POST-TASK[0]")
					So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | POST-TASK[0]")
					<-finished

					Convey("Then the destination file should exist", func() {
						src, err := ioutil.ReadFile("utils.go")
						So(err, ShouldBeNil)
						dst, err := ioutil.ReadFile(root + receive.Path + "/utils.dst")
						So(err, ShouldBeNil)

						So(dst, ShouldResemble, src)
					})

					Convey("Then the server transfer should be over", func() {
						var transfers []model.Transfer
						So(db.Select(&transfers, nil), ShouldBeNil)
						So(transfers, ShouldBeEmpty)

						var results []model.TransferHistory
						So(db.Select(&results, nil), ShouldBeNil)
						So(len(results), ShouldEqual, 2)

						expected := model.TransferHistory{
							ID:             1,
							Owner:          database.Owner,
							IsServer:       !trans.IsServer,
							IsSend:         receive.IsSend,
							Account:        localAccount.Login,
							Agent:          localAgent.Name,
							Protocol:       "sftp",
							SourceFilename: ".",
							DestFilename:   trans.DestPath,
							Rule:           receive.Name,
							Start:          results[0].Start,
							Stop:           results[0].Stop,
							Status:         model.StatusDone,
							Error:          model.TransferError{},
							Step:           "",
							Progress:       0,
							TaskNumber:     0,
						}

						So(results[0], ShouldResemble, expected)
					})

					Convey("Then the client transfer should be over", func() {
						var results []model.TransferHistory
						So(db.Select(&results, nil), ShouldBeNil)
						So(len(results), ShouldEqual, 2)

						expected := model.TransferHistory{
							ID:             2,
							Owner:          database.Owner,
							IsServer:       trans.IsServer,
							IsSend:         send.IsSend,
							Account:        remoteAccount.Login,
							Agent:          remoteAgent.Name,
							Protocol:       "sftp",
							SourceFilename: trans.SourcePath,
							DestFilename:   trans.DestPath,
							Rule:           send.Name,
							Start:          start,
							Stop:           results[1].Stop,
							Status:         model.StatusDone,
							Error:          model.TransferError{},
							Step:           "",
							Progress:       0,
							TaskNumber:     0,
						}

						So(results[1], ShouldResemble, expected)
					})
				})

				Convey("Given that the server pre-tasks fail", func() {
					receivePreTaskFail := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(receivePreTaskFail), ShouldBeNil)

					Convey("When launching the transfer with the client", func() {
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | ERROR-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | ERROR-TASK[0]")
						<-finished

						SkipConvey("Then the destination file should NOT exist", func() {
							_, err := os.Stat("utils.go")
							So(os.IsNotExist(err), ShouldBeTrue)
						})

						Convey("Then the server transfer should be in error", func() {
							var transfers []model.Transfer
							So(db.Select(&transfers, nil), ShouldBeNil)
							So(transfers, ShouldBeEmpty)

							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             1,
								Owner:          database.Owner,
								IsServer:       !trans.IsServer,
								IsSend:         receive.IsSend,
								Account:        localAccount.Login,
								Agent:          localAgent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   trans.DestPath,
								Rule:           receive.Name,
								Start:          results[0].Start,
								Stop:           results[0].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Task TESTFAIL @ receive PRE[1]: task failed",
								},
								Step:       model.StepPreTasks,
								Progress:   0,
								TaskNumber: 1,
							}

							So(results[0], ShouldResemble, expected)
						})

						Convey("Then the client transfer should be in error", func() {
							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             2,
								Owner:          database.Owner,
								IsServer:       trans.IsServer,
								IsSend:         send.IsSend,
								Account:        remoteAccount.Login,
								Agent:          remoteAgent.Name,
								Protocol:       "sftp",
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           send.Name,
								Start:          start,
								Stop:           results[1].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Remote pre-tasks failed",
								},
								Step:       model.StepPreTasks,
								Progress:   0,
								TaskNumber: 0,
							}

							So(results[1], ShouldResemble, expected)
						})
					})
				})

				Convey("Given that the client pre-tasks fail", func() {
					sendPreTaskFail := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPre,
						Rank:   1,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(sendPreTaskFail), ShouldBeNil)

					Convey("When launching the transfer with the client", func() {
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | ERROR-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | ERROR-TASK[0]")
						<-finished

						SkipConvey("Then the destination file should NOT exist", func() {
							_, err := os.Stat("utils.go")
							So(os.IsNotExist(err), ShouldBeTrue)
						})

						Convey("Then the server transfer should be in error", func() {
							var transfers []model.Transfer
							So(db.Select(&transfers, nil), ShouldBeNil)
							So(transfers, ShouldBeEmpty)

							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             1,
								Owner:          database.Owner,
								IsServer:       !trans.IsServer,
								IsSend:         receive.IsSend,
								Account:        localAccount.Login,
								Agent:          localAgent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   trans.DestPath,
								Rule:           receive.Name,
								Start:          results[0].Start,
								Stop:           results[0].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Remote pre-tasks failed",
								},
								Step:       model.StepPreTasks,
								Progress:   0,
								TaskNumber: 0,
							}

							So(results[0], ShouldResemble, expected)
						})

						Convey("Then the client transfer should be in error", func() {
							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             2,
								Owner:          database.Owner,
								IsServer:       trans.IsServer,
								IsSend:         send.IsSend,
								Account:        remoteAccount.Login,
								Agent:          remoteAgent.Name,
								Protocol:       "sftp",
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           send.Name,
								Start:          start,
								Stop:           results[1].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Task TESTFAIL @ send PRE[1]: task failed",
								},
								Step:       model.StepPreTasks,
								Progress:   0,
								TaskNumber: 1,
							}

							So(results[1], ShouldResemble, expected)
						})
					})
				})

				Convey("Given that the server post-tasks fail", func() {
					receivePostTaskFail := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPost,
						Rank:   1,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(receivePostTaskFail), ShouldBeNil)

					Convey("When launching the transfer with the client", func() {
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | POST-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | POST-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | ERROR-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | ERROR-TASK[0]")
						<-finished

						SkipConvey("Then the destination file should NOT exist", func() {
							_, err := os.Stat("utils.go")
							So(os.IsNotExist(err), ShouldBeTrue)
						})

						Convey("Then the server transfer should be in error", func() {
							var transfers []model.Transfer
							So(db.Select(&transfers, nil), ShouldBeNil)
							So(transfers, ShouldBeEmpty)

							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             1,
								Owner:          database.Owner,
								IsServer:       !trans.IsServer,
								IsSend:         receive.IsSend,
								Account:        localAccount.Login,
								Agent:          localAgent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   trans.DestPath,
								Rule:           receive.Name,
								Start:          results[0].Start,
								Stop:           results[0].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Task TESTFAIL @ receive POST[1]: task failed",
								},
								Step:       model.StepPostTasks,
								Progress:   0,
								TaskNumber: 1,
							}

							So(results[0], ShouldResemble, expected)
						})

						Convey("Then the client transfer should be in error", func() {
							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             2,
								Owner:          database.Owner,
								IsServer:       trans.IsServer,
								IsSend:         send.IsSend,
								Account:        remoteAccount.Login,
								Agent:          remoteAgent.Name,
								Protocol:       "sftp",
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           send.Name,
								Start:          start,
								Stop:           results[1].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Remote post-tasks failed",
								},
								Step:       model.StepPostTasks,
								Progress:   0,
								TaskNumber: 0,
							}

							So(results[1], ShouldResemble, expected)
						})
					})
				})

				Convey("Given that the client post-tasks fail", func() {
					sendPostTaskFail := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPost,
						Rank:   1,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(sendPostTaskFail), ShouldBeNil)

					Convey("When launching the transfer with the client", func() {
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | PRE-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | POST-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 1 | ERROR-TASK[0]")
						So(<-checkChannel, ShouldEqual, "TESTCHECK | Rule 2 | ERROR-TASK[0]")
						<-finished

						SkipConvey("Then the destination file should NOT exist", func() {
							_, err := os.Stat("utils.go")
							So(os.IsNotExist(err), ShouldBeTrue)
						})

						Convey("Then the server transfer should be in error", func() {
							var transfers []model.Transfer
							So(db.Select(&transfers, nil), ShouldBeNil)
							So(transfers, ShouldBeEmpty)

							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             1,
								Owner:          database.Owner,
								IsServer:       !trans.IsServer,
								IsSend:         receive.IsSend,
								Account:        localAccount.Login,
								Agent:          localAgent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   trans.DestPath,
								Rule:           receive.Name,
								Start:          results[0].Start,
								Stop:           results[0].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Remote post-tasks failed",
								},
								Step:       model.StepPostTasks,
								Progress:   0,
								TaskNumber: 0,
							}

							So(results[0], ShouldResemble, expected)
						})

						Convey("Then the client transfer should be in error", func() {
							var results []model.TransferHistory
							So(db.Select(&results, nil), ShouldBeNil)
							So(len(results), ShouldEqual, 2)

							expected := model.TransferHistory{
								ID:             2,
								Owner:          database.Owner,
								IsServer:       trans.IsServer,
								IsSend:         send.IsSend,
								Account:        remoteAccount.Login,
								Agent:          remoteAgent.Name,
								Protocol:       "sftp",
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           send.Name,
								Start:          start,
								Stop:           results[1].Stop,
								Status:         model.StatusError,
								Error: model.TransferError{
									Code:    model.TeExternalOperation,
									Details: "Task TESTFAIL @ send POST[1]: task failed",
								},
								Step:       model.StepPostTasks,
								Progress:   0,
								TaskNumber: 1,
							}

							So(results[1], ShouldResemble, expected)
						})
					})
				})
			})
		})
	})
}
