package sftp

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileReader(t *testing.T) {
	Convey("Given a database with a rule, a localAgent and a localAccount", t, func() {
		root := "test_file_reader"
		Reset(func() { _ = os.RemoveAll(root) })
		So(os.MkdirAll(root+"/test", 0700), ShouldBeNil)

		err := ioutil.WriteFile(root+"/test/file.test", []byte("Test file"), 0600)
		So(err, ShouldBeNil)

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "test",
			IsSend: true,
			Path:   "/test",
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"` +
				root + `"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		Convey("Given the Filereader", func() {
			report := make(chan progress, 1)
			handler := makeHandlers(db, agent, account, report).FileGet

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

						Convey("With a valid Source, Desitation and Status", func() {
							So(trans.SourcePath, ShouldEqual, request.Filepath)
							So(trans.DestPath, ShouldEqual, filepath.Base(request.Filepath))
							So(trans.Status, ShouldEqual, model.StatusTransfer)
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
	Convey("Given a database with a rule and a localAgent", t, func() {
		root := "test_file_writer"
		Reset(func() { _ = os.RemoveAll(root) })

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "test",
			IsSend: false,
			Path:   "/test",
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"` +
				root + `"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		Convey("Given the Filewriter", func() {
			report := make(chan progress, 1)
			handler := makeHandlers(db, agent, account, report).FilePut

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

						Convey("With a valid Source, Desitation and Status", func() {
							So(trans.SourcePath, ShouldEqual, filepath.Base(request.Filepath))
							So(trans.DestPath, ShouldEqual, request.Filepath)
							So(trans.Status, ShouldEqual, model.StatusTransfer)
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
