package sftp

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileReader(t *testing.T) {
	Convey("Given a database with a rule, a localAgent and a localAccount", t, func() {
		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:  "test",
			IsGet: true,
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"test_sftp_root"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		Convey("Given the Filereader", func() {
			handler := makeHandlers(db, agent, account).FileGet

			Convey("Given a request for an existing file in the rule path", func() {
				request := &sftp.Request{
					Filepath: "test/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Fileread(request)

					Convey("Then is should return NO error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then a transfer should be present in db", func() {
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
		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:  "test",
			IsGet: false,
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"test_sftp_root"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "test",
		}
		So(db.Create(account), ShouldBeNil)

		Convey("Given the Filewriterr", func() {
			handler := makeHandlers(db, agent, account).FilePut

			Convey("Given a request for an existing file in the rule path", func() {
				request := &sftp.Request{
					Filepath: "test/file.test",
				}

				Convey("When calling the handler", func() {
					_, err := handler.Filewrite(request)

					Convey("Then is should return NO error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then a transfer should be present in db", func() {
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
}
