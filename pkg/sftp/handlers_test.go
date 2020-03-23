package sftp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
)

var Dir string

func init() {
	var err error
	Dir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func TestFileReader(t *testing.T) {
	logger := log.NewLogger("test_file_reader")

	Convey("Given a database with a rule, a localAgent and a localAccount", t, func() {
		root, err := ioutil.TempDir("", "gateway-test")
		So(err, ShouldBeNil)
		So(os.Mkdir(root+"/test", 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		err = ioutil.WriteFile(root+"/test/file.test", []byte("Test file"), 0600)
		So(err, ShouldBeNil)

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:    "test",
			IsSend:  true,
			Path:    "/test",
			OutPath: "/test",
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
			ProtoConfig: []byte(`{"address":"localhost","port":2023}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		var serverConf config.SftpProtoConfig
		So(json.Unmarshal(agent.ProtoConfig, &serverConf), ShouldBeNil)

		Convey("Given the Filereader", func() {
			handler := (&sshListener{
				DB:          db,
				Logger:      logger,
				Agent:       agent,
				ProtoConfig: &serverConf,
				GWConf:      &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: Dir}},
			}).makeFileReader(context.Background(), account.ID)

			Convey("Given a request for an existing file in the rule path", func() {
				request := &sftp.Request{
					Filepath: "/test/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Fileread(request)

					Convey("Then is should return NO error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then a transfer should be present in db", func() {
						trans := &model.Transfer{
							RuleID:    rule.ID,
							IsServer:  true,
							AgentID:   agent.ID,
							AccountID: account.ID,
						}
						So(db.Get(trans), ShouldBeNil)

						Convey("With a valid Source, Destination and Status", func() {
							So(trans.SourceFile, ShouldEqual, filepath.Base(request.Filepath))
							So(trans.DestFile, ShouldEqual, ".")
							So(trans.Status, ShouldEqual, model.StatusRunning)
						})
					})
				})
			})

			Convey("Given a request for an non existing rule", func() {
				request := &sftp.Request{
					Filepath: "toto/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Fileread(request)

					Convey("Then is should return an error", func() {
						So(err, ShouldNotBeNil)
					})
				})
			})

			Convey("Given a request for a file at the server root", func() {
				request := &sftp.Request{
					Filepath: "file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Fileread(request)

					Convey("Then is should return an error", func() {
						So(err, ShouldNotBeNil)
					})
				})
			})
		})
	})
}

func TestFileWriter(t *testing.T) {
	logger := log.NewLogger("test_file_writer")

	Convey("Given a database with a rule and a localAgent", t, func() {
		root := utils.SlashJoin(Dir, "test_file_writer")
		Reset(func() { _ = os.RemoveAll(root) })

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "test",
			IsSend: false,
			Path:   "/test",
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
			ProtoConfig: []byte(`{"address":"localhost","port":2023}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		var serverConf config.SftpProtoConfig
		So(json.Unmarshal(agent.ProtoConfig, &serverConf), ShouldBeNil)

		Convey("Given the Filewriter", func() {
			handler := (&sshListener{
				DB:          db,
				Logger:      logger,
				Agent:       agent,
				ProtoConfig: &serverConf,
				GWConf:      &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: Dir}},
			}).makeFileWriter(context.Background(), account.ID)

			Convey("Given a request for an existing file in the rule path", func() {
				request := &sftp.Request{
					Filepath: "/test/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Filewrite(request)

					Convey("Then is should return NO error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then a transfer should be present in db", func() {
						trans := &model.Transfer{
							RuleID:    rule.ID,
							IsServer:  true,
							AgentID:   agent.ID,
							AccountID: account.ID,
						}
						So(db.Get(trans), ShouldBeNil)

						Convey("With a valid Source, Destination and Status", func() {
							So(trans.SourceFile, ShouldEqual, ".")
							So(trans.DestFile, ShouldEqual, filepath.Base(request.Filepath))
							So(trans.Status, ShouldEqual, model.StatusRunning)
						})
					})
				})
			})

			Convey("Given a request for an non existing rule", func() {
				request := &sftp.Request{
					Filepath: "/toto/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Filewrite(request)

					Convey("Then is should return an error", func() {
						So(err, ShouldNotBeNil)
					})
				})
			})

			Convey("Given a request for a file at the server root", func() {
				request := &sftp.Request{
					Filepath: "file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Filewrite(request)

					Convey("Then is should return an error", func() {
						So(err, ShouldNotBeNil)
					})
				})
			})
		})
	})
}
