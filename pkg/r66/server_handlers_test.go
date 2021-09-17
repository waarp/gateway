package r66

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestValidAuth(t *testing.T) {
	logger := log.NewLogger("test_valid_auth")

	Convey("Given an R66 authentication handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		r66Server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
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
			db:     db,
			logger: logger,
			agent:  r66Server,
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
						So(ses.hasHash, ShouldBeTrue)
						So(ses.hasFileSize, ShouldBeTrue)
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
	logger := log.NewLogger("test_valid_request")

	Convey("Given an R66 authentication handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		rule := &model.Rule{
			Name:   "rule",
			IsSend: false,
			Path:   "/rule",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
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
				db:     db,
				logger: logger,
				agent:  server,
			}},
			account:     account,
			hasHash:     true,
			hasFileSize: true,
		}

		Convey("Given a request packet", func() {
			packet := &r66.Request{
				ID:       1,
				Filepath: "/file",
				FileSize: 4,
				Rule:     rule.Name,
				Mode:     3,
				Block:    512,
				Rank:     0,
				// Limit:      0,
				Infos: nil,
			}

			shouldFailWith := func(desc, msg string) {
				Convey("When calling the `ValidAuth` function", func() {
					_, err := ses.ValidRequest(packet)

					Convey("Then it should return an error saying that "+desc, func() {
						So(err, ShouldBeError, msg)
					})
				})
			}

			Convey("Given that the packet is valid", func() {
				Convey("When calling the `ValidAuth` function", func() {
					t, err := ses.ValidRequest(packet)
					So(err, ShouldBeNil)
					trans, ok := t.(*transferHandler)
					So(ok, ShouldBeTrue)

					Convey("Then it should have created a transfer", func() {
						So(trans.file.Transfer.RuleID, ShouldEqual, rule.ID)
						So(trans.file.Transfer.IsServer, ShouldBeTrue)
						So(trans.file.Transfer.AgentID, ShouldEqual, server.ID)
						So(trans.file.Transfer.AccountID, ShouldEqual, account.ID)
						So(trans.file.Transfer.TrueFilepath, ShouldEqual, packet.Filepath+".tmp")
						So(trans.file.Transfer.SourceFile, ShouldEqual, path.Base(packet.Filepath))
						So(trans.file.Transfer.DestFile, ShouldEqual, path.Base(packet.Filepath))
						So(trans.file.Transfer.Start, ShouldHappenOnOrBefore, time.Now())
						So(trans.file.Transfer.Step, ShouldEqual, types.StepNone)
						So(trans.file.Transfer.Status, ShouldEqual, types.StatusRunning)
					})

					Convey("Then it should return a new session handler", func() {
						So(trans.file.Rule, ShouldResemble, rule)
						So(trans.isMD5, ShouldBeTrue)
						So(trans.fileSize, ShouldEqual, packet.FileSize)
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

			Convey("Given that the file size is missing", func() {
				packet.FileSize = -1
				shouldFailWith("the file size is missing", "n: missing file size")
			})
		})
	})
}

func TestUpdateTransferInfo(t *testing.T) {
	logger := log.NewLogger("test_valid_request")

	Convey("Given an R66 transfer handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		paths := pipeline.Paths{
			PathsConfig: conf.PathsConfig{
				GatewayHome:   testhelpers.TempDir(c, "test_r66_updatetransferinfo"),
				InDirectory:   "gw_in",
				OutDirectory:  "gw_out",
				WorkDirectory: "gw_tmp",
			},
			ServerRoot: "serv_root",
			ServerIn:   "serv_in",
			ServerOut:  "serv_out",
			ServerWork: "serv_tmp",
		}

		send := &model.Rule{Name: "send", IsSend: true, Path: "/send"}
		So(db.Insert(send).Run(), ShouldBeNil)
		recv := &model.Rule{Name: "recv", IsSend: false, Path: "/recv"}
		So(db.Insert(recv).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
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
				IsServer:         true,
				AgentID:          server.ID,
				AccountID:        account.ID,
				SourceFile:       "old.file",
				DestFile:         "old.file",
				Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
				Step:             types.StepPreTasks,
				Status:           types.StatusRunning,
				Owner:            database.Owner,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			pip, err := pipeline.NewTransferStream(context.Background(), logger, db,
				paths, trans)
			So(err, ShouldBeNil)
			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{Service: &Service{
						db:     db,
						logger: logger,
						agent:  server,
					}},
				},
				file: &stream{
					TransferStream: pip,
				},
				isMD5:    false,
				fileSize: 100,
			}

			Convey("When calling the 'UpdateTransferInfo' handler", func() {
				info := &r66.UpdateInfo{
					Filename: "new.file",
					FileSize: 200,
					FileInfo: &r66.TransferData{},
				}
				So(hand.UpdateTransferInfo(info), ShouldBeNil)

				var check model.Transfer
				So(db.Get(&check, "id=?", trans.ID).Run(), ShouldBeNil)

				Convey("Then it should have updated the transfer's filename", func() {
					So(check.SourceFile, ShouldEqual, "new.file")
					So(check.DestFile, ShouldEqual, "new.file")
					So(filepath.Base(check.TrueFilepath), ShouldEqual, "new.file")
				})

				Convey("Then it should have updated the transfer's file size", func() {
					So(hand.fileSize, ShouldEqual, 200)
				})
			})
		})

		Convey("Given a pull transfer", func() {
			trans := &model.Transfer{
				RemoteTransferID: "1",
				RuleID:           send.ID,
				IsServer:         true,
				AgentID:          server.ID,
				AccountID:        account.ID,
				SourceFile:       "new.file",
				DestFile:         "new.file",
				Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
				Step:             types.StepPreTasks,
				Status:           types.StatusRunning,
				Owner:            database.Owner,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			dir := filepath.Join(paths.GatewayHome, paths.ServerRoot, paths.ServerOut)
			So(os.MkdirAll(dir, 0o700), ShouldBeNil)
			So(ioutil.WriteFile(filepath.Join(dir, "new.file"), []byte("file content"), 0o600), ShouldBeNil)
			pip, err := pipeline.NewTransferStream(context.Background(), logger, db,
				paths, trans)
			So(err, ShouldBeNil)
			hand := transferHandler{
				sessionHandler: &sessionHandler{
					authHandler: &authHandler{Service: &Service{
						db:     db,
						logger: logger,
						agent:  server,
					}},
				},
				file: &stream{
					TransferStream: pip,
				},
				isMD5:    false,
				fileSize: 100,
			}

			Convey("When calling the 'UpdateTransferInfo' handler", func() {
				info := &r66.UpdateInfo{}
				So(hand.UpdateTransferInfo(info), ShouldBeNil)

				var check model.Transfer
				So(db.Get(&check, "id=?", trans.ID).Run(), ShouldBeNil)

				Convey("Then it should have updated the transfer's filename", func() {
					So(info.Filename, ShouldEqual, "new.file")
					So(check.DestFile, ShouldEqual, "new.file")
					So(filepath.Base(check.TrueFilepath), ShouldEqual, "new.file")
				})

				Convey("Then it should have updated the transfer's file size", func() {
					So(info.FileSize, ShouldEqual, 12)
					So(hand.fileSize, ShouldEqual, 12)
				})
			})
		})
	})
}
