package r66

import (
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

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
						ses := s.(*sessionHandler)
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
	logger := log.NewLogger("test_valid_request")

	Convey("Given an R66 authentication handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		root := testhelpers.TempDir(c, "r66_valid_request")

		rule := &model.Rule{
			Name:        "rule",
			IsSend:      false,
			Path:        "/rule",
			LocalTmpDir: "rule_tmp",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
			Root:        filepath.Join(root, "server_root"),
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
				agent:            server,
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
				Filepath: "/file",
				FileSize: 4,
				Rule:     rule.Name,
				IsRecv:   false,
				IsMD5:    true,
				Block:    512,
				Rank:     0,
				//Limit:      0,
				Infos: "",
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
					handler := t.(*transferHandler)

					Convey("Then it should have created a transfer", func() {
						So(handler.trans.pip.TransCtx.Transfer.RuleID, ShouldEqual, rule.ID)
						So(handler.trans.pip.TransCtx.Transfer.IsServer, ShouldBeTrue)
						So(handler.trans.pip.TransCtx.Transfer.AgentID, ShouldEqual, server.ID)
						So(handler.trans.pip.TransCtx.Transfer.AccountID, ShouldEqual, account.ID)
						So(handler.trans.pip.TransCtx.Transfer.LocalPath, ShouldEqual, filepath.Join(
							server.Root, rule.LocalTmpDir, path.Base(packet.Filepath)))
						So(handler.trans.pip.TransCtx.Transfer.RemotePath, ShouldEqual, "/"+path.Base(packet.Filepath))
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

func hash(pwd string) []byte {
	crypt := r66.CryptPass([]byte(pwd))
	h, err := bcrypt.GenerateFromPassword(crypt, bcrypt.MinCost)
	So(err, ShouldBeNil)
	return h
}
