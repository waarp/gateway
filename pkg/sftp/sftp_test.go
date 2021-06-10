package sftp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSFTPPackage(t *testing.T) {
	logger := log.NewLogger("test_sftp_package")

	Convey("Given a gateway root", t, func(c C) {
		home := testhelpers.TempDir(c, "package_test_root")

		pathConf := conf.PathsConfig{
			GatewayHome:   home,
			InDirectory:   home,
			OutDirectory:  home,
			WorkDirectory: filepath.Join(home, "tmp"),
		}

		Convey("Given an SFTP server", func(dbc C) {
			listener, err := net.Listen("tcp", "localhost:0")
			So(err, ShouldBeNil)
			_, port, err := net.SplitHostPort(listener.Addr().String())
			So(err, ShouldBeNil)

			root := filepath.Join(home, "sftp_root")
			So(os.Mkdir(root, 0o700), ShouldBeNil)

			db := database.TestDatabase(dbc, "ERROR")
			localAgent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Root:        root,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:" + port,
			}
			So(db.Insert(localAgent).Run(), ShouldBeNil)
			var protoConfig config.SftpProtoConfig
			So(json.Unmarshal(localAgent.ProtoConfig, &protoConfig), ShouldBeNil)

			pwd := "tata"
			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "toto",
				Password:     []byte(pwd),
			}
			So(db.Insert(localAccount).Run(), ShouldBeNil)

			localServerCert := model.Cert{
				OwnerType:   localAgent.TableName(),
				OwnerID:     localAgent.ID,
				Name:        "test_sftp_server_cert",
				PrivateKey:  testPK,
				PublicKey:   testPBK,
				Certificate: []byte("cert"),
			}
			So(db.Insert(&localServerCert).Run(), ShouldBeNil)

			localUserCert := &model.Cert{
				OwnerType:   localAccount.TableName(),
				OwnerID:     localAccount.ID,
				Name:        "test_sftp_user_cert",
				PublicKey:   []byte(rsaPBK),
				Certificate: []byte{'.'},
			}
			So(db.Insert(localUserCert).Run(), ShouldBeNil)

			receive := &model.Rule{
				Name:     "receive",
				Comment:  "",
				IsSend:   false,
				Path:     "receive_path",
				InPath:   "receive/in",
				OutPath:  "receive/out",
				WorkPath: "receive/work",
			}
			So(db.Insert(receive).Run(), ShouldBeNil)

			receivePreTask := &model.Task{
				RuleID: receive.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   "TESTCHECK",
				Args:   []byte(`{"msg":"RECEIVE | PRE-TASK[0] | OK"}`),
			}
			receivePostTask := &model.Task{
				RuleID: receive.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   "TESTCHECK",
				Args:   []byte(`{"msg":"RECEIVE | POST-TASK[0] | OK"}`),
			}
			receiveErrorTask := &model.Task{
				RuleID: receive.ID,
				Chain:  model.ChainError,
				Rank:   0,
				Type:   "TESTCHECK",
				Args:   []byte(`{"msg":"RECEIVE | ERROR-TASK[0] | OK"}`),
			}
			So(db.Insert(receivePreTask).Run(), ShouldBeNil)
			So(db.Insert(receivePostTask).Run(), ShouldBeNil)
			So(db.Insert(receiveErrorTask).Run(), ShouldBeNil)

			serverConfig, err := getSSHServerConfig(db, []model.Cert{localServerCert},
				&protoConfig, localAgent)
			So(err, ShouldBeNil)
			ctx, cancel := context.WithCancel(context.Background())

			sshList := &sshListener{
				DB:          db,
				Logger:      logger,
				Agent:       localAgent,
				ProtoConfig: &protoConfig,
				GWConf:      &conf.ServerConfig{Paths: pathConf},
				SSHConf:     serverConfig,
				Listener:    listener,
				connWg:      sync.WaitGroup{},
				ctx:         ctx,
				cancel:      cancel,
			}
			sshList.listen()
			sshList.handlerMaker = sshList.makeTestHandlers
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = sshList.close(ctx)
			})

			Convey("Given an SFTP client", func() {
				remoteAgent := &model.RemoteAgent{
					Name:        "test_sftp_partner",
					Protocol:    "sftp",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:" + port,
				}
				So(db.Insert(remoteAgent).Run(), ShouldBeNil)

				remoteAccount := &model.RemoteAccount{
					RemoteAgentID: remoteAgent.ID,
					Login:         "toto",
					Password:      []byte(pwd),
				}
				So(db.Insert(remoteAccount).Run(), ShouldBeNil)

				remoteServerCert := &model.Cert{
					OwnerType:   remoteAgent.TableName(),
					OwnerID:     remoteAgent.ID,
					Name:        "test_sftp_partner_cert",
					PublicKey:   testPBK,
					Certificate: []byte("cert"),
				}
				So(db.Insert(remoteServerCert).Run(), ShouldBeNil)

				remoteUserCert := &model.Cert{
					OwnerType:   remoteAccount.TableName(),
					OwnerID:     remoteAccount.ID,
					Name:        "test_sftp_account_cert",
					PrivateKey:  []byte(rsaPK),
					Certificate: []byte{'.'},
				}
				So(db.Insert(remoteUserCert).Run(), ShouldBeNil)

				send := &model.Rule{
					Name:    "send",
					Comment: "",
					IsSend:  true,
					Path:    "send_path",
					InPath:  receive.Path,
					OutPath: "send/out",
				}
				So(db.Insert(send).Run(), ShouldBeNil)

				sendPreTask := &model.Task{
					RuleID: send.ID,
					Chain:  model.ChainPre,
					Rank:   0,
					Type:   "TESTCHECK",
					Args:   []byte(`{"msg":"SEND | PRE-TASK[0] | OK"}`),
				}
				sendPostTask := &model.Task{
					RuleID: send.ID,
					Chain:  model.ChainPost,
					Rank:   0,
					Type:   "TESTCHECK",
					Args:   []byte(`{"msg":"SEND | POST-TASK[0] | OK"}`),
				}
				sendErrorTask := &model.Task{
					RuleID: send.ID,
					Chain:  model.ChainError,
					Rank:   0,
					Type:   "TESTCHECK",
					Args:   []byte(`{"msg":"SEND | ERROR-TASK[0] | OK"}`),
				}
				So(db.Insert(sendPreTask).Run(), ShouldBeNil)
				So(db.Insert(sendPostTask).Run(), ShouldBeNil)
				So(db.Insert(sendErrorTask).Run(), ShouldBeNil)

				Convey("Given a transfer from SFTP client to server", func() {
					srcFile := "sftp_test_file.src"
					content := make([]byte, 1048576)
					_, err := rand.Read(content)
					So(err, ShouldBeNil)
					srcFilepath := filepath.Join(home, send.OutPath, srcFile)
					So(os.MkdirAll(filepath.Dir(srcFilepath), 0o700), ShouldBeNil)
					So(ioutil.WriteFile(srcFilepath, content, 0o600), ShouldBeNil)

					trans := model.Transfer{
						RuleID:     send.ID,
						IsServer:   false,
						AgentID:    remoteAgent.ID,
						AccountID:  remoteAccount.ID,
						SourceFile: srcFile,
						DestFile:   "sftp_test_file.dst",
						Start:      time.Now(),
						Status:     types.StatusPlanned,
					}
					So(db.Insert(&trans).Run(), ShouldBeNil)

					Convey("Given an executor", func() {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()

						paths := pipeline.Paths{PathsConfig: pathConf}
						stream, err := pipeline.NewTransferStream(ctx, logger,
							db, paths, &trans)
						So(err, ShouldBeNil)

						exe := executor.Executor{
							TransferStream: stream,
							Ctx:            ctx,
						}

						Convey("Given that the transfer is successful", func() {
							exe.Run()
							checkChannel <- "END TRANSFER 1"

							Convey("When launching the transfer with the client", func() {
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "RECEIVE | POST-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END SERVER TRANSFER")
								So(<-checkChannel, ShouldEqual, "END TRANSFER 1")

								Convey("Then the destination file should exist", func() {
									file := filepath.Join(root, receive.InPath, trans.DestFile)
									dst, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(len(dst), ShouldEqual, len(content))
									So(dst, ShouldResemble, content)
								})

								Convey("Then the transfers should be over", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).Run(), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									var hist model.Histories
									So(db.Select(&hist).OrderBy("id", true).Run(), ShouldBeNil)
									So(hist, ShouldHaveLength, 2)

									Convey("Then there should be a client-side "+
										"history entry", func() {
										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       false,
											IsSend:         true,
											Account:        remoteAccount.Login,
											Agent:          remoteAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          hist[0].Start,
											Stop:           hist[0].Stop,
											Status:         types.StatusDone,
											Step:           types.StepNone,
											Error:          types.TransferError{},
											Progress:       uint64(len(content)),
											TaskNumber:     0,
										}
										So(hist[0], ShouldResemble, expected)
									})

									Convey("Then there should be a server-side "+
										"history entry", func() {
										expected := model.TransferHistory{
											ID:             2,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
											Rule:           receive.Name,
											Start:          hist[1].Start,
											Stop:           hist[1].Stop,
											Status:         types.StatusDone,
											Step:           types.StepNone,
											Error:          types.TransferError{},
											Progress:       uint64(len(content)),
											TaskNumber:     0,
										}
										So(hist[1], ShouldResemble, expected)
									})
								})
							})
						})

						Convey("Given that the server pre-tasks fail", func() {
							receivePreTaskFail := &model.Task{
								RuleID: receive.ID,
								Chain:  model.ChainPre,
								Rank:   1,
								Type:   "TESTFAIL",
								Args:   []byte(`{"msg":"RECEIVE | PRE-TASK[1] | FAIL"}`),
							}
							So(db.Insert(receivePreTaskFail).Run(), ShouldBeNil)

							exe.Run()
							checkChannel <- "END TRANSFER 2"

							Convey("When launching the transfer with the client", func() {
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[1] | FAIL")
								So(<-checkChannel, ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END TRANSFER 2")

								Convey("Then the work file should NOT exist", func() {
									file := filepath.Join(root, receive.WorkPath,
										trans.DestFile+".tmp")
									_, err := os.Stat(file)
									So(os.IsNotExist(err), ShouldBeTrue)
								})

								Convey("Then the transfers should be over", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).OrderBy("id", true).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 2)

									Convey("Then there should be a client-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        1,
											Owner:     database.Owner,
											IsServer:  false,
											AccountID: remoteAccount.ID,
											AgentID:   remoteAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												home, send.OutPath, trans.SourceFile)),
											SourceFile: trans.SourceFile,
											DestFile:   trans.DestFile,
											RuleID:     send.ID,
											Start:      transfers[0].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code: types.TeExternalOperation,
												Details: "Remote pre-tasks failed: Task " +
													"TESTFAIL @ receive PRE[1]: task failed",
											},
											Step:       types.StepSetup,
											Progress:   0,
											TaskNumber: 0,
										}
										So(transfers[0], ShouldResemble, expected)
									})

									Convey("Then there should be a server-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        2,
											Owner:     database.Owner,
											IsServer:  true,
											AccountID: localAccount.ID,
											AgentID:   localAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												root, receive.WorkPath, trans.DestFile+".tmp")),
											SourceFile: trans.DestFile,
											DestFile:   trans.DestFile,
											RuleID:     receive.ID,
											Start:      transfers[1].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code: types.TeExternalOperation,
												Details: "Task TESTFAIL @ receive " +
													"PRE[1]: task failed",
											},
											Step:       types.StepPreTasks,
											Progress:   0,
											TaskNumber: 1,
										}
										So(transfers[1], ShouldResemble, expected)
									})
								})
							})
						})

						Convey("Given that the client pre-tasks fail", func() {
							sendPreTaskFail := &model.Task{
								RuleID: send.ID,
								Chain:  model.ChainPre,
								Rank:   1,
								Type:   "TESTFAIL",
								Args:   []byte(`{"msg":"SEND | PRE-TASK[1] | FAIL"}`),
							}
							So(db.Insert(sendPreTaskFail).Run(), ShouldBeNil)

							exe.Run()
							checkChannel <- "END TRANSFER 3"

							Convey("When launching the transfer with the client", func() {
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASK[1] | FAIL")
								So(<-checkChannel, ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END SERVER TRANSFER")
								So(<-checkChannel, ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END TRANSFER 3")

								Convey("Then the work file should NOT exist", func() {
									file := filepath.Join(root, receive.WorkPath,
										trans.DestFile+".tmp")
									_, err := os.Stat(file)
									So(os.IsNotExist(err), ShouldBeTrue)
								})

								Convey("Then the transfers should be over", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).OrderBy("id", true).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 2)

									Convey("Then there should be a client-side"+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        1,
											Owner:     database.Owner,
											IsServer:  trans.IsServer,
											AccountID: remoteAccount.ID,
											AgentID:   remoteAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												home, send.OutPath, trans.SourceFile)),
											SourceFile: trans.SourceFile,
											DestFile:   trans.DestFile,
											RuleID:     send.ID,
											Start:      transfers[0].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code: types.TeExternalOperation,
												Details: "Task TESTFAIL @ send " +
													"PRE[1]: task failed",
											},
											Step:       types.StepPreTasks,
											Progress:   0,
											TaskNumber: 1,
										}
										So(transfers[0], ShouldResemble, expected)
									})

									Convey("Then there should be a server-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        2,
											Owner:     database.Owner,
											IsServer:  true,
											AccountID: localAccount.ID,
											AgentID:   localAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												root, receive.WorkPath, trans.DestFile+".tmp")),
											SourceFile: trans.DestFile,
											DestFile:   trans.DestFile,
											RuleID:     receive.ID,
											Start:      transfers[1].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code:    types.TeExternalOperation,
												Details: "Remote pre-tasks failed",
											},
											Step:       types.StepPreTasks,
											Progress:   0,
											TaskNumber: 1,
										}
										So(transfers[1], ShouldResemble, expected)
									})
								})
							})
						})

						Convey("Given that the server post-tasks fail", func() {
							receivePostTaskFail := &model.Task{
								RuleID: receive.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTFAIL",
								Args:   []byte(`{"msg":"RECEIVE | POST-TASK[1] | FAIL"}`),
							}
							So(db.Insert(receivePostTaskFail).Run(), ShouldBeNil)

							exe.Run()
							checkChannel <- "END TRANSFER 4"

							Convey("When launching the transfer with the client", func() {
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "RECEIVE | POST-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "RECEIVE | POST-TASK[1] | FAIL")
								So(<-checkChannel, ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END SERVER TRANSFER")
								So(<-checkChannel, ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END TRANSFER 4")

								Convey("Then the file should exist", func() {
									file := filepath.Join(root, receive.InPath,
										trans.DestFile)
									cont, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(len(cont), ShouldEqual, len(content))
									So(cont, ShouldResemble, content)
								})

								Convey("Then the transfers should be over", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).OrderBy("id", true).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 2)

									Convey("Then there should be a client-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        1,
											Owner:     database.Owner,
											IsServer:  false,
											AccountID: remoteAccount.ID,
											AgentID:   remoteAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												home, send.OutPath, trans.SourceFile)),
											SourceFile: trans.SourceFile,
											DestFile:   trans.DestFile,
											RuleID:     send.ID,
											Start:      transfers[0].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code: types.TeExternalOperation,
												Details: "Remote post-tasks failed: Task " +
													"TESTFAIL @ receive POST[1]: task failed",
											},
											Step:       types.StepFinalization,
											Progress:   uint64(len(content)),
											TaskNumber: 0,
										}
										So(transfers[0], ShouldResemble, expected)
									})

									Convey("Then there should be a server-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        2,
											Owner:     database.Owner,
											IsServer:  true,
											AccountID: localAccount.ID,
											AgentID:   localAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												root, receive.InPath, trans.DestFile)),
											SourceFile: trans.DestFile,
											DestFile:   trans.DestFile,
											RuleID:     receive.ID,
											Start:      transfers[1].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code: types.TeExternalOperation,
												Details: "Task TESTFAIL @ receive " +
													"POST[1]: task failed",
											},
											Step:       types.StepPostTasks,
											Progress:   uint64(len(content)),
											TaskNumber: 1,
										}
										So(transfers[1], ShouldResemble, expected)
									})
								})
							})
						})

						Convey("Given that the client post-tasks fail", func() {
							sendPostTaskFail := &model.Task{
								RuleID: send.ID,
								Chain:  model.ChainPost,
								Rank:   1,
								Type:   "TESTFAIL",
								Args:   []byte(`{"msg":"SEND | POST-TASK[1] | FAIL"}`),
							}
							So(db.Insert(sendPostTaskFail).Run(), ShouldBeNil)

							exe.Run()
							checkChannel <- "END TRANSFER 5"

							Convey("When launching the transfer with the client", func() {
								So(<-checkChannel, ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "SEND | POST-TASK[1] | FAIL")
								So(<-checkChannel, ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END SERVER TRANSFER")
								So(<-checkChannel, ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(<-checkChannel, ShouldEqual, "END TRANSFER 5")

								Convey("Then the file should exist", func() {
									file := filepath.Join(root, receive.WorkPath,
										trans.DestFile+".tmp")
									cont, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(len(cont), ShouldEqual, len(content))
									So(cont, ShouldResemble, content)
								})

								Convey("Then the transfers should be over", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).OrderBy("id", true).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 2)

									Convey("Then there should be a client-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        1,
											Owner:     database.Owner,
											IsServer:  false,
											AccountID: remoteAccount.ID,
											AgentID:   remoteAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												home, send.OutPath, trans.SourceFile)),
											SourceFile: trans.SourceFile,
											DestFile:   trans.DestFile,
											RuleID:     send.ID,
											Start:      transfers[0].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code:    types.TeExternalOperation,
												Details: "Task TESTFAIL @ send POST[1]: task failed",
											},
											Step:       types.StepPostTasks,
											Progress:   uint64(len(content)),
											TaskNumber: 1,
										}
										So(transfers[0], ShouldResemble, expected)
									})

									Convey("Then there should be a server-side "+
										"transfer entry in error", func() {
										expected := model.Transfer{
											ID:        2,
											Owner:     database.Owner,
											IsServer:  true,
											AccountID: localAccount.ID,
											AgentID:   localAgent.ID,
											TrueFilepath: utils.NormalizePath(filepath.Join(
												root, receive.WorkPath, trans.DestFile+".tmp")),
											SourceFile: trans.DestFile,
											DestFile:   trans.DestFile,
											RuleID:     receive.ID,
											Start:      transfers[1].Start,
											Status:     types.StatusError,
											Error: types.TransferError{
												Code:    types.TeConnectionReset,
												Details: "SFTP connection closed unexpectedly",
											},
											Step:       types.StepData,
											Progress:   uint64(len(content)),
											TaskNumber: 0,
										}
										So(transfers[1], ShouldResemble, expected)
									})
								})
							})
						})
					})
				})
			})
		})
	})
}
