package r66

import (
	"context"
	"path"
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestValidAuth(t *testing.T) {
	Convey("Given an R66 authentication handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_valid_auth")
		db := database.TestDatabase(c)
		r66Server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l"},
			Address:     types.Addr("localhost", 0),
		}
		So(db.Insert(r66Server).Run(), ShouldBeNil)

		toto := &model.LocalAccount{
			LocalAgentID: r66Server.ID,
			Login:        "toto",
		}
		So(db.Insert(toto).Run(), ShouldBeNil)

		totoPswd := &model.Credential{
			LocalAccountID: utils.NewNullInt64(toto.ID),
			Type:           auth.Password,
			Value:          "sesame",
		}
		So(db.Insert(totoPswd).Run(), ShouldBeNil)

		handler := &authHandler{service: &service{
			db:      db,
			logger:  logger,
			agent:   r66Server,
			r66Conf: &serverConfig{},
		}}

		Convey("Given an authentication packet", func() {
			packet := &r66.Authent{
				Login:     toto.Login,
				Password:  r66.CryptPass([]byte("sesame")),
				Address:   "1.2.3.4:6666",
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

				shouldFailWith("the credentials are incorrect", "A: authentication failed")
			})

			Convey("Given an incorrect password", func() {
				packet.Password = []byte("not sesame")

				shouldFailWith("the credentials are incorrect", "A: authentication failed")
			})

			Convey("Given an incorrect hash digest", func() {
				packet.Digest = "BAD"

				shouldFailWith("the credentials are incorrect", "A: unsuported hash algorithm")
			})

			Convey("Given that the account is IP-restricted", func() {
				toto.IPAddresses = []string{"1.2.3.4"}
				So(db.Update(toto).Run(), ShouldBeNil)

				Convey("When logging in from the correct IP", func() {
					packet.Address = "1.2.3.4:6666"

					Convey("Then it should succeed", func() {
						_, err := handler.ValidAuth(packet)
						So(err, ShouldBeNil)
					})
				})

				Convey("When logging in from an unauthorized IP", func() {
					packet.Address = "5.6.7.8:6666"

					Convey("Then it should fail", func() {
						_, err := handler.ValidAuth(packet)
						So(err, ShouldBeError, "A: unauthorized IP address")
					})
				})
			})
		})
	})
}

func TestValidRequest(t *testing.T) {
	root := t.TempDir()

	Convey("Given an R66 session handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_valid_request")
		db := database.TestDatabase(c)

		rule := &model.Rule{
			Name:           "rule",
			IsSend:         false,
			Path:           "/rule",
			TmpLocalRcvDir: "rule_tmp",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l"},
			Address:     types.Addr("localhost", 0),
			RootDir:     path.Join(root, "server_root"),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		accPswd := &model.Credential{
			LocalAccountID: utils.NewNullInt64(account.ID),
			Type:           auth.Password,
			Value:          "sesame",
		}
		So(db.Insert(accPswd).Run(), ShouldBeNil)

		ses := sessionHandler{
			authHandler: &authHandler{service: &service{
				db:     db,
				logger: logger,
				agent:  server,
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
					defer handler.trans.pip.EndTransfer()

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

						handler, ok := t.(*transferHandler)
						So(ok, ShouldBeTrue)
						defer handler.trans.pip.EndTransfer()
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
	root := t.TempDir()

	Convey("Given an R66 transfer handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_valid_request")
		db := database.TestDatabase(c)
		conf.GlobalConfig.Paths = conf.PathsConfig{
			GatewayHome: root,
		}

		send := &model.Rule{Name: "send", IsSend: true, LocalDir: "send_dir"}
		So(db.Insert(send).Run(), ShouldBeNil)

		recv := &model.Rule{Name: "recv", IsSend: false, LocalDir: "recv_dir", TmpLocalRcvDir: "recv_tmp"}
		So(db.Insert(recv).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l"},
			Address:     types.Addr("localhost", 0),
			RootDir:     "server_root",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		accPswd := &model.Credential{
			LocalAccountID: utils.NewNullInt64(account.ID),
			Type:           auth.Password,
			Value:          "sesame",
		}
		So(db.Insert(accPswd).Run(), ShouldBeNil)

		Convey("Given a push transfer", func() {
			trans := &model.Transfer{
				RemoteTransferID: "1",
				RuleID:           recv.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				DestFilename:     "old.file",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			pip, err := pipeline.NewServerPipeline(db, logger, trans, nil)
			So(err, ShouldBeNil)
			defer pip.EndTransfer()

			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)

			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{service: &service{
						db:     db,
						logger: logger,
						agent:  server,
					}},
				},
				trans:  &serverTransfer{pip: pip, ctx: ctx},
				cancel: cancel,
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
					So(check.LocalPath, ShouldEqual, fs.JoinPath(root,
						server.RootDir, recv.TmpLocalRcvDir, info.Filename))
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

			dir := path.Join(root, server.RootDir, send.LocalDir)
			So(fs.MkdirAll(dir), ShouldBeNil)
			So(fs.WriteFullFile(path.Join(dir, "new.file"),
				[]byte("file content")), ShouldBeNil)

			pip, err := pipeline.NewServerPipeline(db, logger, trans, nil)
			So(err, ShouldBeNil)
			defer pip.EndTransfer()

			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)

			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{
						service: &service{
							db:     db,
							logger: logger,
							agent:  server,
						},
					},
				},
				trans:  &serverTransfer{pip: pip, ctx: ctx},
				cancel: cancel,
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
