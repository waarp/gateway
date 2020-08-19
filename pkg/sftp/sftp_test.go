package sftp

import (
	"context"
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
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSFTPPackage(t *testing.T) {
	logger := log.NewLogger("test_sftp_package")

	Convey("Given a gateway root", t, func() {
		home, err := filepath.Abs("package_test_root")
		So(err, ShouldBeNil)
		So(os.Mkdir(home, 0o700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(home) })

		pathConf := conf.PathsConfig{
			GatewayHome:   home,
			InDirectory:   home,
			OutDirectory:  home,
			WorkDirectory: filepath.Join(home, "tmp"),
		}

		Convey("Given an SFTP server", func() {
			listener, err := net.Listen("tcp", "localhost:0")
			So(err, ShouldBeNil)
			_, port, err := net.SplitHostPort(listener.Addr().String())
			So(err, ShouldBeNil)

			root := filepath.Join(home, "sftp_root")
			So(os.Mkdir(root, 0o700), ShouldBeNil)

			db := database.GetTestDatabase()
			localAgent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Paths:       &model.ServerPaths{Root: root},
				ProtoConfig: []byte(`{"address":"localhost","port":` + port + `}`),
			}
			So(db.Create(localAgent), ShouldBeNil)
			var protoConfig config.SftpProtoConfig
			So(json.Unmarshal(localAgent.ProtoConfig, &protoConfig), ShouldBeNil)

			pwd := "tata"
			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "toto",
				Password:     []byte(pwd),
			}
			So(db.Create(localAccount), ShouldBeNil)

			localServerCert := &model.Cert{
				OwnerType:   localAgent.TableName(),
				OwnerID:     localAgent.ID,
				Name:        "test_sftp_server_cert",
				PrivateKey:  testPK,
				PublicKey:   testPBK,
				Certificate: []byte("cert"),
			}
			So(db.Create(localServerCert), ShouldBeNil)

			localUserCert := &model.Cert{
				OwnerType:   localAccount.TableName(),
				OwnerID:     localAccount.ID,
				Name:        "test_sftp_user_cert",
				PublicKey:   []byte(rsaPBK),
				Certificate: []byte{'.'},
			}
			So(db.Create(localUserCert), ShouldBeNil)

			receive := &model.Rule{
				Name:     "receive",
				Comment:  "",
				IsSend:   false,
				Path:     "rule_path",
				InPath:   "receive/in",
				OutPath:  "receive/out",
				WorkPath: "receive/work",
			}
			So(db.Create(receive), ShouldBeNil)

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
			So(db.Create(receivePreTask), ShouldBeNil)
			So(db.Create(receivePostTask), ShouldBeNil)
			So(db.Create(receiveErrorTask), ShouldBeNil)

			serverConfig, err := getSSHServerConfig(db, localServerCert,
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
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = sshList.close(ctx)
			})

			Convey("Given an SFTP client", func() {
				remoteAgent := &model.RemoteAgent{
					Name:        "test_sftp_partner",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":` + port + `}`),
				}
				So(db.Create(remoteAgent), ShouldBeNil)

				remoteAccount := &model.RemoteAccount{
					RemoteAgentID: remoteAgent.ID,
					Login:         "toto",
					Password:      []byte(pwd),
				}
				So(db.Create(remoteAccount), ShouldBeNil)

				remoteServerCert := &model.Cert{
					OwnerType:   remoteAgent.TableName(),
					OwnerID:     remoteAgent.ID,
					Name:        "test_sftp_partner_cert",
					PublicKey:   testPBK,
					Certificate: []byte("cert"),
				}
				So(db.Create(remoteServerCert), ShouldBeNil)

				remoteUserCert := &model.Cert{
					OwnerType:   remoteAccount.TableName(),
					OwnerID:     remoteAccount.ID,
					Name:        "test_sftp_account_cert",
					PrivateKey:  []byte(rsaPK),
					Certificate: []byte{'.'},
				}
				So(db.Create(remoteUserCert), ShouldBeNil)

				send := &model.Rule{
					Name:    "send",
					Comment: "",
					IsSend:  true,
					Path:    "rule_path",
					InPath:  "send/in",
					OutPath: "send/out",
				}
				So(db.Create(send), ShouldBeNil)

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
				So(db.Create(sendPreTask), ShouldBeNil)
				So(db.Create(sendPostTask), ShouldBeNil)
				So(db.Create(sendErrorTask), ShouldBeNil)

				Convey("Given a transfer from SFTP client to server", func() {
					srcFile := "sftp_test_file.src"
					content := []byte("SFTP package test file content")
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
						Start:      time.Now().Truncate(time.Second),
						Status:     model.StatusPlanned,
					}
					So(db.Create(&trans), ShouldBeNil)

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
						finished := make(chan struct{})

						Convey("Given that the transfer is successful", func() {
							go func() {
								exe.Run()
								close(finished)
							}()

							Convey("When launching the transfer with the client", func() {
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "RECEIVE | POST-TASK[0] | OK")
								So(waitChannel(finished), ShouldBeNil)

								Convey("Then the destination file should exist", func() {
									file := filepath.Join(root, receive.InPath, trans.DestFile)
									dst, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(string(dst), ShouldResemble, string(content))
								})

								Convey("Then the transfers should be over", func() {
									var transfers []model.Transfer
									So(db.Select(&transfers, nil), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									Convey("Then there should be a server-side "+
										"history entry", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
											Rule:           receive.Name,
											Start:          results[0].Start,
											Stop:           results[0].Stop,
											Status:         model.StatusDone,
											Error:          model.TransferError{},
											Progress:       0,
											TaskNumber:     0,
										}
										So(results[0], ShouldResemble, expected)
									})

									Convey("Then there should be a client-side "+
										"history entry", func() {
										var hist []model.TransferHistory
										So(db.Select(&hist, nil), ShouldBeNil)
										So(len(hist), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             2,
											Owner:          database.Owner,
											IsServer:       false,
											IsSend:         true,
											Account:        remoteAccount.Login,
											Agent:          remoteAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          hist[1].Start,
											Stop:           hist[1].Stop,
											Status:         model.StatusDone,
											Error:          model.TransferError{},
											Progress:       0,
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
							So(db.Create(receivePreTaskFail), ShouldBeNil)

							go func() {
								exe.Run()
								close(finished)
							}()

							Convey("When launching the transfer with the client", func() {
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[1] | FAIL")
								So(getNextTask(), ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(waitChannel(finished), ShouldBeNil)

								Convey("Then the work file should NOT exist", func() {
									file := filepath.Join(root, receive.WorkPath,
										trans.DestFile+".tmp")
									_, err := os.Stat(file)
									So(os.IsNotExist(err), ShouldBeTrue)
								})

								Convey("Then the transfers should be over", func() {
									var transfers []model.Transfer
									So(db.Select(&transfers, nil), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									Convey("Then there should be a server-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
											Rule:           receive.Name,
											Start:          results[0].Start,
											Stop:           results[0].Stop,
											Status:         model.StatusError,
											Error: model.TransferError{
												Code: model.TeExternalOperation,
												Details: "Task TESTFAIL @ receive " +
													"PRE[1]: task failed",
											},
											Step:       model.StepPreTasks,
											Progress:   0,
											TaskNumber: 1,
										}

										So(results[0], ShouldResemble, expected)
									})

									Convey("Then there should be a client-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             2,
											Owner:          database.Owner,
											IsServer:       false,
											IsSend:         true,
											Account:        remoteAccount.Login,
											Agent:          remoteAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          results[1].Start,
											Stop:           results[1].Stop,
											Status:         model.StatusError,
											Error: model.TransferError{
												Code: model.TeExternalOperation,
												Details: "Remote pre-tasks failed: Task " +
													"TESTFAIL @ receive PRE[1]: task failed",
											},
											Step:       model.StepPreTasks,
											Progress:   0,
											TaskNumber: 0,
										}

										So(results[1], ShouldResemble, expected)
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
							So(db.Create(sendPreTaskFail), ShouldBeNil)

							go func() {
								exe.Run()
								close(finished)
							}()

							Convey("When launching the transfer with the client", func() {
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | PRE-TASK[1] | FAIL")
								So(getNextTask(), ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(waitChannel(finished), ShouldBeNil)

								Convey("Then the work file should NOT exist", func() {
									file := filepath.Join(root, receive.WorkPath,
										trans.DestFile+".tmp")
									_, err := os.Stat(file)
									So(os.IsNotExist(err), ShouldBeTrue)
								})

								Convey("Then the transfers should be over", func() {
									var transfers []model.Transfer
									So(db.Select(&transfers, nil), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									Convey("Then there should be a server-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
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

									Convey("Then there should be a client-side"+
										"history entry in error", func() {
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
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          results[1].Start,
											Stop:           results[1].Stop,
											Status:         model.StatusError,
											Error: model.TransferError{
												Code: model.TeExternalOperation,
												Details: "Task TESTFAIL @ send " +
													"PRE[1]: task failed",
											},
											Step:       model.StepPreTasks,
											Progress:   0,
											TaskNumber: 1,
										}

										So(results[1], ShouldResemble, expected)
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
							So(db.Create(receivePostTaskFail), ShouldBeNil)

							go func() {
								exe.Run()
								close(finished)
							}()

							Convey("When launching the transfer with the client", func() {
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "RECEIVE | POST-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "RECEIVE | POST-TASK[1] | FAIL")
								So(getNextTask(), ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(waitChannel(finished), ShouldBeNil)

								Convey("Then the file should exist", func() {
									file := filepath.Join(root, receive.InPath,
										trans.DestFile)
									cont, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(string(cont), ShouldEqual, string(content))
								})

								Convey("Then the transfers should be over", func() {
									var transfers []model.Transfer
									So(db.Select(&transfers, nil), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									Convey("Then there should be a server-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
											Rule:           receive.Name,
											Start:          results[0].Start,
											Stop:           results[0].Stop,
											Status:         model.StatusError,
											Error: model.TransferError{
												Code: model.TeExternalOperation,
												Details: "Task TESTFAIL @ receive " +
													"POST[1]: task failed",
											},
											Step:       model.StepPostTasks,
											Progress:   0,
											TaskNumber: 1,
										}

										So(results[0], ShouldResemble, expected)
									})

									Convey("Then there should be a client-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             2,
											Owner:          database.Owner,
											IsServer:       false,
											IsSend:         true,
											Account:        remoteAccount.Login,
											Agent:          remoteAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          results[1].Start,
											Stop:           results[1].Stop,
											Status:         model.StatusError,
											Error: model.TransferError{
												Code: model.TeExternalOperation,
												Details: "Remote post-tasks failed: Task " +
													"TESTFAIL @ receive POST[1]: task failed",
											},
											Step:       model.StepPostTasks,
											Progress:   0,
											TaskNumber: 0,
										}

										So(results[1], ShouldResemble, expected)
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
							So(db.Create(sendPostTaskFail), ShouldBeNil)

							go func() {
								exe.Run()
								close(finished)
							}()

							Convey("When launching the transfer with the client", func() {
								So(getNextTask(), ShouldEqual, "RECEIVE | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | PRE-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | POST-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | POST-TASK[1] | FAIL")
								So(getNextTask(), ShouldEqual, "RECEIVE | ERROR-TASK[0] | OK")
								So(getNextTask(), ShouldEqual, "SEND | ERROR-TASK[0] | OK")
								So(waitChannel(finished), ShouldBeNil)

								Convey("Then the file should exist", func() {
									file := filepath.Join(root, receive.InPath,
										trans.DestFile)
									cont, err := ioutil.ReadFile(file)
									So(err, ShouldBeNil)

									So(string(cont), ShouldEqual, string(content))
								})

								Convey("Then the transfers should be over", func() {
									var transfers []model.Transfer
									So(db.Select(&transfers, nil), ShouldBeNil)
									So(transfers, ShouldBeEmpty)

									Convey("Then there should be a server-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             1,
											Owner:          database.Owner,
											IsServer:       true,
											IsSend:         false,
											Account:        localAccount.Login,
											Agent:          localAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.DestFile,
											DestFilename:   trans.DestFile,
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

									Convey("Then there should be a client-side "+
										"history entry in error", func() {
										var results []model.TransferHistory
										So(db.Select(&results, nil), ShouldBeNil)
										So(len(results), ShouldEqual, 2)

										expected := model.TransferHistory{
											ID:             2,
											Owner:          database.Owner,
											IsServer:       false,
											IsSend:         true,
											Account:        remoteAccount.Login,
											Agent:          remoteAgent.Name,
											Protocol:       "sftp",
											SourceFilename: trans.SourceFile,
											DestFilename:   trans.DestFile,
											Rule:           send.Name,
											Start:          results[1].Start,
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
			})
		})
	})
}
