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
		args := map[string]string{
			"file": "/test/file",
			"to":   "partner",
			"as":   "account",
			"rule": "transfer rule",
		}

		Convey("Given that the arguments are valid", func() {
			b, err := json.Marshal(args)
			So(err, ShouldBeNil)

			task := &model.Task{
				Args: b,
			}

			Convey("When validating the task", func() {
				err := trans.Validate(task)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a parameter is missing", func() {
			delete(args, "to")
			b, err := json.Marshal(args)
			So(err, ShouldBeNil)

			task := &model.Task{
				Args: b,
			}

			Convey("When validating the task", func() {
				err := trans.Validate(task)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is empty", func() {
			args["rule"] = ""
			b, err := json.Marshal(args)
			So(err, ShouldBeNil)

			task := &model.Task{
				Args: b,
			}

			Convey("When validating the task", func() {
				err := trans.Validate(task)

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
			Name: "transfer rule",
			Path: "/rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		partner := &model.RemoteAgent{
			Name:        "test partner",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"port":1,"address":"localhost","root":"/root"}`),
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
			processor := &Processor{Db: db}

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
							So(msg, ShouldEqual, "error getting partner: the "+
								"record does not exist")
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
							So(msg, ShouldEqual, "error getting account: the "+
								"record does not exist")
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
							So(msg, ShouldEqual, "error getting rule: the "+
								"record does not exist")
						})
					})
				})
			})
		})
	})
}
