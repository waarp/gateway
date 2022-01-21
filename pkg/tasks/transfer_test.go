package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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
	Convey("Given a coherent database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		rule := &model.Rule{
			Name:   "transfer rule",
			IsSend: true,
			Path:   "rule/path",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		partner := &model.RemoteAgent{
			Name:        "test partner",
			Protocol:    testProtocol,
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
		}
		So(db.Insert(partner).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: partner.ID,
			Login:         "test account",
			Password:      "sesame",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		Convey("Given a 'TRANSFER' task", func() {
			trans := &TransferTask{}
			args := map[string]string{
				"file": "/test/file",
				"to":   partner.Name,
				"as":   account.Login,
				"rule": rule.Name,
			}

			Convey("Given that the parameters are valid", func() {
				Convey("When running the task", func() {
					msg, err := trans.Run(context.Background(), args, db, nil)

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
					msg, err := trans.Run(context.Background(), args, db, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'to' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, fmt.Sprintf(
								"failed to retrieve partner '%s': partner not found",
								args["to"]))
						})
					})
				})
			})

			Convey("Given that the account does not exist", func() {
				args["as"] = "toto"

				Convey("When running the task", func() {
					msg, err := trans.Run(context.Background(), args, db, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'as' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, fmt.Sprintf(
								"failed to retrieve account '%s': remote account not found",
								args["as"]))
						})
					})
				})
			})

			Convey("Given that the rule does not exist", func() {
				args["rule"] = "toto"

				Convey("When running the task", func() {
					msg, err := trans.Run(context.Background(), args, db, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)

						Convey("Then the log message should say that the 'rule' "+
							"parameter is invalid", func() {
							So(msg, ShouldEqual, fmt.Sprintf(
								"failed to retrieve rule '%s': rule not found",
								args["rule"]))
						})
					})
				})
			})
		})
	})
}
