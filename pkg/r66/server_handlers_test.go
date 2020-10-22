package r66

import (
	"path"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"
)

func TestValidAuth(t *testing.T) {
	logger := log.NewLogger("test_valid_auth")

	Convey("Given an R66 authentication handler", t, func(c C) {
		db := database.GetTestDatabase()
		r66Server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
		}
		So(db.Create(r66Server), ShouldBeNil)

		toto := &model.LocalAccount{
			LocalAgentID: r66Server.ID,
			Login:        "toto",
			Password:     []byte("sesame"),
		}
		So(db.Create(toto), ShouldBeNil)

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
		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "rule",
			IsSend: false,
			Path:   "/rule",
		}
		So(db.Create(rule), ShouldBeNil)

		server := &model.LocalAgent{
			Name:        "r66 server",
			Protocol:    "r66",
			ProtoConfig: []byte(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:6666",
		}
		So(db.Create(server), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
			Password:     []byte("sesame"),
		}
		So(db.Create(account), ShouldBeNil)

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
				//Limit:      0,
				TransferInfo: nil,
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
					trans := t.(*transferHandler)

					Convey("Then it should have created a transfer", func() {
						So(trans.stream.Transfer.RuleID, ShouldEqual, rule.ID)
						So(trans.stream.Transfer.IsServer, ShouldBeTrue)
						So(trans.stream.Transfer.AgentID, ShouldEqual, server.ID)
						So(trans.stream.Transfer.AccountID, ShouldEqual, account.ID)
						So(trans.stream.Transfer.TrueFilepath, ShouldEqual, packet.Filepath+".tmp")
						So(trans.stream.Transfer.SourceFile, ShouldEqual, path.Base(packet.Filepath))
						So(trans.stream.Transfer.DestFile, ShouldEqual, path.Base(packet.Filepath))
						So(trans.stream.Transfer.Start, ShouldHappenOnOrBefore, time.Now())
						So(trans.stream.Transfer.Step, ShouldEqual, model.StepNone)
						So(trans.stream.Transfer.Status, ShouldEqual, model.StatusRunning)
					})

					Convey("Then it should return a new session handler", func() {
						So(trans.stream.Rule, ShouldResemble, rule)
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
				packet.FileSize = 0
				shouldFailWith("the file size is missing", "n: missing file size")
			})
		})
	})
}
