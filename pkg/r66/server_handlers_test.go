package r66

import (
	"path"
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestValidAuth(t *testing.T) {
	Convey("Given an R66 authentication handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_valid_auth")
		db := database.TestDatabase(c)
		r66Server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    ProtocolR66,
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
		}
		So(db.Insert(r66Server).Run(), ShouldBeNil)

		toto := &model.LocalAccount{
			LocalAgentID: r66Server.ID,
			Login:        "toto",
			PasswordHash: hash("sesame"),
		}
		So(db.Insert(toto).Run(), ShouldBeNil)

		handler := &authHandler{Service: &Service{
			db:      db,
			logger:  logger,
			agentID: r66Server.ID,
		}}

		Convey("Given an authentication packet", func() {
			packet := &r66.Authent{
				Login:     "toto",
				Password:  r66.CryptPass([]byte("sesame")),
				Filesize:  true,
				FinalHash: true,
				Digest:    "SHA-256",
			}

			shouldFailWith := func(desc, msg string) {
				Convey("When calling the `ValidAuth` function", func() {
					_, err := handler.ValidAuth(packet)

					Convey("Then it should return an error saying that "+desc, func() {
						So(err, ShouldBeError, msg)
					})
				})
			}

			Convey("Given that the packet is valid", func() {
				Convey("When calling the `ValidAuth` function", func() {
					s, err := handler.ValidAuth(packet)
					So(err, ShouldBeNil)

					Convey("Then it should return a new session handler", func() {
						ses, ok := s.(*sessionHandler)
						So(ok, ShouldBeTrue)
						So(ses.account, ShouldResemble, toto)
						So(ses.conf.FinalHash, ShouldBeTrue)
						So(ses.conf.Filesize, ShouldBeTrue)
					})
				})
			})

			Convey("Given an incorrect login", func() {
				packet.Login = "tata"

				shouldFailWith("the credentials are incorrect", "A: incorrect credentials")
			})

			Convey("Given an incorrect password", func() {
				packet.Password = []byte("not sesame")

				shouldFailWith("the credentials are incorrect", "A: incorrect credentials")
			})

			Convey("Given an incorrect hash digest", func() {
				packet.Digest = "SHA-512"

				shouldFailWith("the digest is invalid", "U: unknown final hash digest")
			})
		})
	})
}

func TestValidRequest(t *testing.T) {
	Convey("Given an R66 session handler", t, func(c C) {
		fstest.InitMemFS(c)

		logger := testhelpers.TestLogger(c, "test_valid_request")
		db := database.TestDatabase(c)
		root := "memory:/r66_valid_request"

		rule := &model.Rule{
			Name:           "rule",
			IsSend:         false,
			Path:           "/rule",
			TmpLocalRcvDir: "rule_tmp",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    ProtocolR66,
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
			RootDir:     path.Join(root, "server_root"),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
			PasswordHash: hash("sesame"),
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		ses := sessionHandler{
			authHandler: &authHandler{Service: &Service{
				db:               db,
				logger:           logger,
				agentID:          server.ID,
				runningTransfers: service.NewTransferMap(),
			}},
			account: account,
			conf: &r66.Authent{
				FinalHash: true,
				Filesize:  true,
			},
		}

		Convey("Given a request packet", func() {
			packet := &r66.Request{
				ID:       1,
				Filepath: "file.ex",
				FileSize: 4,
				Rule:     rule.Name,
				IsRecv:   false,
				IsMD5:    true,
				Block:    512,
				Rank:     0,
				Infos:    "",
			}

			shouldFailWith := func(desc, msg string) {
				Convey("When calling the `ValidRequest` function", func() {
					_, err := ses.ValidRequest(packet)

					Convey("Then it should return an error saying that "+desc, func() {
						So(err, ShouldBeError, msg)
					})
				})
			}

			Convey("Given that the packet is valid", func() {
				Convey("When calling the `ValidRequest` function", func() {
					t, err := ses.ValidRequest(packet)
					So(err, ShouldBeNil)

					handler, ok := t.(*transferHandler)
					So(ok, ShouldBeTrue)

					Convey("Then it should have created a transfer", func() {
						So(handler.trans.pip.TransCtx.Transfer.RuleID, ShouldEqual, rule.ID)
						So(handler.trans.pip.TransCtx.Transfer.LocalAccountID.Int64, ShouldEqual, account.ID)
						So(handler.trans.pip.TransCtx.Transfer.DestFilename, ShouldEqual, packet.Filepath)
						So(handler.trans.pip.TransCtx.Transfer.Start, ShouldHappenOnOrBefore, time.Now())
						So(handler.trans.pip.TransCtx.Transfer.Step, ShouldEqual, types.StepSetup)
						So(handler.trans.pip.TransCtx.Transfer.Status, ShouldEqual, types.StatusRunning)
					})

					Convey("Then it should have returned a new session handler", func() {
						So(handler.trans.pip.TransCtx.Rule, ShouldResemble, rule)
						So(handler.trans.pip.TransCtx.LocalAgent, ShouldResemble, server)
						So(handler.trans.pip.TransCtx.LocalAccount, ShouldResemble, account)
					})
				})
			})

			Convey("Given that the file size is missing", func() {
				packet.FileSize = model.UnknownSize

				Convey("When calling the `ValidRequest` function", func() {
					t, err := ses.ValidRequest(packet)

					Convey("Then it should return NO error", func() {
						So(err, ShouldBeNil)

						_, ok := t.(*transferHandler)
						So(ok, ShouldBeTrue)
					})
				})
			})

			Convey("Given that the filename is missing", func() {
				packet.Filepath = ""

				shouldFailWith("the filename is missing", "n: missing filepath")
			})

			Convey("Given that the rule name is invalid", func() {
				packet.Rule = "tata"

				shouldFailWith("the rule could not be found", "n: rule does not exist")
			})

			Convey("Given that the block size is missing", func() {
				packet.Block = 0

				shouldFailWith("the block size is missing", "n: missing block size")
			})
		})
	})
}

func TestUpdateTransferInfo(t *testing.T) {
	Convey("Given an R66 transfer handler", t, func(c C) {
		testFS := fstest.InitMemFS(c)
		logger := testhelpers.TestLogger(c, "test_valid_request")
		db := database.TestDatabase(c)
		root := "memory:/r66_update_info"
		conf.GlobalConfig.Paths = conf.PathsConfig{
			GatewayHome: root,
		}

		send := &model.Rule{Name: "send", IsSend: true, LocalDir: "send_dir"}
		So(db.Insert(send).Run(), ShouldBeNil)

		recv := &model.Rule{Name: "recv", IsSend: false, LocalDir: "recv_dir", TmpLocalRcvDir: "recv_tmp"}
		So(db.Insert(recv).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    ProtocolR66,
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
			RootDir:     "server_root",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
			PasswordHash: hash("sesame"),
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		Convey("Given a push transfer", func() {
			trans := &model.Transfer{
				RemoteTransferID: "1",
				RuleID:           recv.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				DestFilename:     "old.file",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			pip, err := pipeline.NewServerPipeline(db, trans)
			So(err, ShouldBeNil)

			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{Service: &Service{
						db:      db,
						logger:  logger,
						agentID: server.ID,
					}},
				},
				trans: &serverTransfer{
					pip:   pip,
					store: utils.NewErrorStorage(),
				},
			}

			Convey("When calling the 'UpdateTransferInfo' handler", func() {
				info := &r66.UpdateInfo{
					Filename: "new.file",
					FileSize: 200,
					FileInfo: &r66.TransferData{},
				}
				So(hand.UpdateTransferInfo(info), ShouldBeNil)

				check := pip.TransCtx.Transfer

				Convey("Then it should have updated the transfer's filename", func() {
					So(check.LocalPath.String(), ShouldEqual, mkURL(root,
						server.RootDir, recv.TmpLocalRcvDir, info.Filename).String())
				})

				Convey("Then it should have updated the transfer's file size", func() {
					So(check.Filesize, ShouldEqual, 200)
				})
			})
		})

		Convey("Given a pull transfer", func() {
			trans := &model.Transfer{
				RemoteTransferID: "1",
				RuleID:           send.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				SrcFilename:      "new.file",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			dir := mkURL(root, server.RootDir, send.LocalDir)
			So(fs.MkdirAll(testFS, dir), ShouldBeNil)
			So(fs.WriteFullFile(testFS, dir.JoinPath("new.file"),
				[]byte("file content")), ShouldBeNil)

			pip, err := pipeline.NewServerPipeline(db, trans)
			So(err, ShouldBeNil)

			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{
						Service: &Service{
							db:      db,
							logger:  logger,
							agentID: server.ID,
						},
					},
				},
				trans: &serverTransfer{
					pip:   pip,
					store: utils.NewErrorStorage(),
				},
			}

			Convey("When calling the 'RunPreTask' handler", func() {
				info, err := hand.RunPreTask()
				So(err, ShouldBeNil)

				Convey("Then it should have returned the transfer's filename", func() {
					So(info.Filename, ShouldEqual, "new.file")
				})

				Convey("Then it should have returned the transfer's file size", func() {
					So(info.FileSize, ShouldEqual, 12)
				})
			})
		})
	})
}
