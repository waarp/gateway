package tasks

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTransferValidate(t *testing.T) {

	Convey("Given a 'TRANSFER' task", t, func() {
		trans := &TransferTask{}

		Convey("Given that the arguments are valid", func() {
			args := map[string]string{
				"file": "/test/file",
				"to":   "partner",
				"as":   "account",
				"rule": "transfer rule",
			}

			Convey("When validating the task", func() {
				err := trans.Validate(args)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a parameter is missing", func() {
			args := map[string]string{
				"file": "/test/file",
				"as":   "account",
				"rule": "transfer rule",
			}

			Convey("When validating the task", func() {
				err := trans.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is empty", func() {
			args := map[string]string{
				"file": "/test/file",
				"to":   "partner",
				"as":   "account",
				"rule": "",
			}

			Convey("When validating the task", func() {
				err := trans.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestTransferRun(t *testing.T) {

	Convey("Given a coherent database", t, func() {
		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "transfer rule",
			IsSend: true,
			Path:   "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		partner := &model.RemoteAgent{
			Name:        "test partner",
			Protocol:    "sftp",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
		}
		So(db.Create(partner), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   partner.TableName(),
			OwnerID:     partner.ID,
			Name:        "test cert",
			PublicKey:   []byte("public key"),
			Certificate: []byte("certificate"),
		}
		So(db.Create(cert), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: partner.ID,
			Login:         "test account",
			Password:      []byte("password"),
		}
		So(db.Create(account), ShouldBeNil)

		Convey("Given a 'TRANSFER' task", func() {
			trans := &TransferTask{}
			args := map[string]string{
				"file": "/test/file",
				"to":   partner.Name,
				"as":   account.Login,
				"rule": rule.Name,
			}
			processor := &Processor{DB: db}

			Convey("Given that the parameters are valid", func() {

				Convey("When running the task", func() {
					msg, err := trans.Run(args, processor)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)

						Convey("Then the log message should be empty", func() {
							So(msg, ShouldBeEmpty)
						})
					})
				})
			})

			Convey("Given that the partner does not exist", func() {
				args["to"] = "toto"

				Convey("When running the task", func() {
					msg, err := trans.Run(args, processor)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'to' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, "failed to retrieve partner '"+
								args["to"]+"': the record does not exist")
						})
					})
				})
			})

			Convey("Given that the account does not exist", func() {
				args["as"] = "toto"

				Convey("When running the task", func() {
					msg, err := trans.Run(args, processor)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'as' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, "failed to retrieve account '"+
								args["as"]+"': the record does not exist")
						})
					})
				})
			})

			Convey("Given that the rule does not exist", func() {
				args["rule"] = "toto"

				Convey("When running the task", func() {
					msg, err := trans.Run(args, processor)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'rule' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, "failed to retrieve rule '"+
								args["rule"]+"': the record does not exist")
						})
					})
				})
			})
		})
	})
}
