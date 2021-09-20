package sftp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestFileReader(t *testing.T) {
	logger := log.NewLogger("test_file_reader")

	Convey("Given a file", t, func(c C) {
		root := testhelpers.TempDir(c, "file_reader_test_root")

		rulePath := filepath.Join(root, "test", "out")
		So(os.MkdirAll(rulePath, 0o700), ShouldBeNil)

		file := filepath.Join(rulePath, "file_read.src")
		content := []byte("File reader test file content")
		So(ioutil.WriteFile(file, content, 0o600), ShouldBeNil)

		Convey("Given a database with a rule, a localAgent and a localAccount", func(dbc C) {
			db := database.TestDatabase(dbc, "ERROR")

			rule := &model.Rule{
				Name:    "test",
				IsSend:  true,
				Path:    "test/path",
				OutPath: "test/out",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Root:        root,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2023",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				PasswordHash: hash("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

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
						_, err := handler.Fileread(request)

						Convey("Then is should return NO error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND is_server=? AND agent_id=? AND account_id=?",
								rule.ID, true, agent.ID, account.ID).Run(), ShouldBeNil)

							Convey("With a valid Source, Destination and Status", func() {
								So(trans.SourceFile, ShouldEqual, filepath.Base(request.Filepath))
								So(trans.DestFile, ShouldEqual, filepath.Base(request.Filepath))
								So(trans.Status, ShouldEqual, types.StatusRunning)
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

	Convey("Given a file", t, func(c C) {
		root := testhelpers.TempDir(c, "file_writer_test_root")

		Convey("Given a database with a rule and a localAgent", func(dbc C) {
			db := database.TestDatabase(dbc, "ERROR")

			rule := &model.Rule{
				Name:   "test",
				IsSend: false,
				Path:   "test",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Root:        root,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2023",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				PasswordHash: hash("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

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
						_, err := handler.Filewrite(request)

						Convey("Then is should return NO error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND is_server=? AND agent_id=? AND account_id=?",
								rule.ID, true, agent.ID, account.ID).Run(), ShouldBeNil)

							Convey("With a valid Source, Destination and Status", func() {
								So(trans.SourceFile, ShouldEqual, filepath.Base(request.Filepath))
								So(trans.DestFile, ShouldEqual, filepath.Base(request.Filepath))
								So(trans.Status, ShouldEqual, types.StatusRunning)
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
