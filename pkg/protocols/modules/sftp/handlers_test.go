package sftp

import (
	"testing"

	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestFileReader(t *testing.T) {
	root := t.TempDir()

	Convey("Given a file", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_file_reader")

		rulePath := fs.JoinPath(root, "test", "out")
		So(fs.MkdirAll(rulePath), ShouldBeNil)

		filePath := fs.JoinPath(rulePath, "file_read.src")
		content := []byte("File reader test file content")
		So(fs.WriteFullFile(filePath, content), ShouldBeNil)

		Convey("Given a database with a rule, a localAgent and a localAccount", func(dbc C) {
			db := database.TestDatabase(dbc)

			rule := &model.Rule{
				Name:     "test",
				IsSend:   true,
				Path:     "test/path",
				LocalDir: "test/out",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name: "test_sftp_server", Protocol: SFTP,
				RootDir: root, Address: types.Addr("localhost", 2023),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			pswd := &model.Credential{
				LocalAccountID: utils.NewNullInt64(account.ID),
				Type:           auth.Password,
				Value:          "password",
			}
			So(db.Insert(pswd).Run(), ShouldBeNil)

			Convey("Given the FileReader", func() {
				conf.GlobalConfig.Paths = conf.PathsConfig{GatewayHome: root}

				handler := (&sshListener{
					DB:     db,
					Logger: logger,
					Server: agent,
				}).makeFileReader(account)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "test/path/file_read.src",
					}

					Convey("When calling the handler", func() {
						f, err := handler.Fileread(request)
						So(err, ShouldBeNil)

						file, ok := f.(*serverPipeline)
						So(ok, ShouldBeTrue)

						defer file.Close()

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND local_account_id=?",
								rule.ID, account.ID).Run(), ShouldBeNil)

							Convey("Then the transfer should have a valid file and status", func() {
								trans := file.pipeline.TransCtx.Transfer

								So(trans.LocalPath, ShouldEqual, fs.JoinPath(
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
	root := t.TempDir()

	Convey("Given a file", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_file_writer")

		Convey("Given a database with a rule and a localAgent", func(c C) {
			db := database.TestDatabase(c)

			rule := &model.Rule{
				Name:   "test_rule",
				IsSend: false,
				Path:   "/test/path",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			agent := &model.LocalAgent{
				Name: "test_sftp_server", Protocol: SFTP,
				RootDir: root, Address: types.Addr("localhost", 2023),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			pswd := &model.Credential{
				LocalAccountID: utils.NewNullInt64(account.ID),
				Type:           auth.Password,
				Value:          "password",
			}
			So(db.Insert(pswd).Run(), ShouldBeNil)

			Convey("Given the Filewriter", func() {
				conf.GlobalConfig.Paths = conf.PathsConfig{GatewayHome: root}

				handler := (&sshListener{
					DB:     db,
					Logger: logger,
					Server: agent,
				}).makeFileWriter(account)

				Convey("Given a request for an existing file in the rule path", func() {
					request := &sftp.Request{
						Filepath: "test/path/file.test",
					}

					Convey("When calling the handler", func() {
						f, err := handler.Filewrite(request)
						So(err, ShouldBeNil)

						file, ok := f.(*serverPipeline)
						So(ok, ShouldBeTrue)

						defer file.Close()

						Convey("Then a transfer should be present in db", func() {
							trans := &model.Transfer{}
							So(db.Get(trans, "rule_id=? AND local_account_id=?",
								rule.ID, account.ID).Run(), ShouldBeNil)

							Convey("Then the transfer should have a valid file and status", func() {
								trans := file.pipeline.TransCtx.Transfer

								So(trans.LocalPath, ShouldEqual, fs.JoinPath(
									root, agent.TmpReceiveDir, "file.test.part"))
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
