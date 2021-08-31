package sftp

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
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
				Name:     "test",
				IsSend:   true,
				Path:     "test/path",
				LocalDir: "test/out",
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
				Login:        "toto",
				PasswordHash: hash("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			var serverConf config.SftpProtoConfig
			So(json.Unmarshal(agent.ProtoConfig, &serverConf), ShouldBeNil)

			Convey("Given the FileReader", func() {
				conf.GlobalConfig.Paths = conf.PathsConfig{GatewayHome: root}

				handler := (&sshListener{
					DB:               db,
					Logger:           logger,
					Agent:            agent,
					ProtoConfig:      &serverConf,
					runningTransfers: service.NewTransferMap(),
				}).makeFileReader(nil, account)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "test/path/file_read.src",
					}

					Convey("When calling the handler", func() {
						f, err := handler.Fileread(request)
						So(err, ShouldBeNil)
						Reset(func() { _ = f.(io.Closer).Close() })

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND is_server=? AND agent_id=? AND account_id=?",
								rule.ID, true, agent.ID, account.ID).Run(), ShouldBeNil)

							Convey("With a valid file and status", func() {
								So(trans.LocalPath, ShouldEqual, filepath.Join(
									root, rule.LocalDir, "file_read.src"))
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
				Name:   "test_rule",
				IsSend: false,
				Path:   "/test_rule",
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
				Login:        "toto",
				PasswordHash: hash("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			var serverConf config.SftpProtoConfig
			So(json.Unmarshal(agent.ProtoConfig, &serverConf), ShouldBeNil)

			Convey("Given the Filewriter", func() {
				conf.GlobalConfig.Paths = conf.PathsConfig{GatewayHome: root}

				handler := (&sshListener{
					DB:               db,
					Logger:           logger,
					Agent:            agent,
					ProtoConfig:      &serverConf,
					runningTransfers: service.NewTransferMap(),
				}).makeFileWriter(nil, account)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "/test_rule/file.dst",
					}

					Convey("When calling the handler", func() {
						f, err := handler.Filewrite(request)
						So(err, ShouldBeNil)
						Reset(func() { _ = f.(io.Closer).Close() })

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND is_server=? AND agent_id=? AND account_id=?",
								rule.ID, true, agent.ID, account.ID).Run(), ShouldBeNil)

							Convey("With a valid file and status", func() {
								So(trans.LocalPath, ShouldEqual, filepath.Join(
									root, agent.LocalTmpDir, "file.dst.part"))
								So(trans.Status, ShouldEqual, types.StatusRunning)
							})
						})
					})
				})

				Convey("Given a request for an non existing rule", func() {
					request := &sftp.Request{
						Filepath: "/toto/file.dst",
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
						Filepath: "file.dst",
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
