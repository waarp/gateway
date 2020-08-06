package sftp

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func TestFileReader(t *testing.T) {
	logger := log.NewLogger("test_file_reader")

	Convey("Given a file", t, func() {
		root, err := filepath.Abs("file_reader_test_root")
		So(err, ShouldBeNil)
		rulePath := filepath.Join(root, "test", "out")
		So(os.MkdirAll(rulePath, 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		file := filepath.Join(rulePath, "file_read.src")
		content := []byte("File reader test file content")
		So(ioutil.WriteFile(file, content, 0600), ShouldBeNil)

		Convey("Given a database with a rule, a localAgent and a localAccount", func() {
			db := database.GetTestDatabase()

			rule := &model.Rule{
				Name:    "test",
				IsSend:  true,
				Path:    "test/path",
				OutPath: "test/out",
			}
			So(db.Create(rule), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Paths:       &model.ServerPaths{Root: root},
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

			Convey("Given the FileReader", func() {
				pathsConfig := conf.PathsConfig{GatewayHome: root}
				paths := &pipeline.Paths{PathsConfig: pathsConfig}

				handler := (&sshListener{
					DB:          db,
					Logger:      logger,
					Agent:       agent,
					ProtoConfig: &serverConf,
					GWConf:      &conf.ServerConfig{Paths: pathsConfig},
				}).makeFileReader(context.Background(), account.ID, paths)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "/test/path/file_read.src",
					}

					Convey("When calling the handler", func() {
						r, err := handler.Fileread(request)
						Reset(func() { _ = r.(io.Closer).Close() })

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
								So(trans.DestFile, ShouldEqual, filepath.Base(request.Filepath))
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
	})
}

func TestFileWriter(t *testing.T) {
	logger := log.NewLogger("test_file_writer")

	Convey("Given a file", t, func() {
		root, err := filepath.Abs("file_writer_test_root")
		So(err, ShouldBeNil)
		So(os.Mkdir(root, 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		Convey("Given a database with a rule and a localAgent", func() {
			db := database.GetTestDatabase()

			rule := &model.Rule{
				Name:   "test",
				IsSend: false,
				Path:   "test",
			}
			So(db.Create(rule), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Paths:       &model.ServerPaths{Root: root},
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
				pathsConfig := conf.PathsConfig{GatewayHome: root}
				paths := &pipeline.Paths{PathsConfig: pathsConfig}

				handler := (&sshListener{
					DB:          db,
					Logger:      logger,
					Agent:       agent,
					ProtoConfig: &serverConf,
					GWConf:      &conf.ServerConfig{Paths: pathsConfig},
				}).makeFileWriter(context.Background(), account.ID, paths)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "/test/file.test",
					}

					Convey("When calling the handler", func() {
						w, err := handler.Filewrite(request)
						Reset(func() { _ = w.(io.Closer).Close() })

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
	})
}
