package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTransferTableName(t *testing.T) {
	Convey("Given a `Transfer` instance", t, func() {
		trans := &Transfer{}

		Convey("When calling the 'TableName' method", func() {
			name := trans.TableName()

			Convey("Then it should return the name of the transfers table", func() {
				So(name, ShouldEqual, TableTransfers)
			})
		})
	})
}

func TestTransferBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains a valid remote agent", func() {
			remote := RemoteAgent{
				Name:        "remote",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(&remote).Run(), ShouldBeNil)

			account := RemoteAccount{
				RemoteAgentID: remote.ID,
				Login:         "toto",
				Password:      "password",
			}
			So(db.Insert(&account).Run(), ShouldBeNil)

			rule := Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			Convey("Given a new transfer", func() {
				trans := Transfer{
					RuleID:       rule.ID,
					IsServer:     false,
					AgentID:      remote.ID,
					AccountID:    account.ID,
					TrueFilepath: "/filepath",
					SourceFile:   "source",
					DestFile:     "dest",
					Start:        time.Now(),
					Status:       "PLANNED",
					Owner:        conf.GlobalConfig.GatewayName,
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := trans.BeforeWrite(db)

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new transfer is valid", func() {

					Convey("When calling the 'BeforeWrite' function", func() {
						So(trans.BeforeWrite(db), ShouldBeNil)

						Convey("Then the transfer status should be 'planned'", func() {
							So(trans.Status, ShouldEqual, "PLANNED")
						})

						Convey("Then the transfer owner should be 'test_gateway'", func() {
							So(trans.Owner, ShouldEqual, "test_gateway")
						})
					})
				})

				Convey("Given that the rule ID is missing", func() {
					trans.RuleID = 0
					shouldFailWith("the rule ID is missing", database.NewValidationError(
						"the transfer's rule ID cannot be empty"))
				})

				Convey("Given that the remote ID is missing", func() {
					trans.AgentID = 0
					shouldFailWith("the remote ID is missing", database.NewValidationError(
						"the transfer's remote ID cannot be empty"))
				})

				Convey("Given that the account ID is missing", func() {
					trans.AccountID = 0
					shouldFailWith("the account ID is missing", database.NewValidationError(
						"the transfer's account ID cannot be empty"))
				})

				Convey("Given that the source is missing", func() {
					trans.SourceFile = ""
					shouldFailWith("the source is missing", database.NewValidationError(
						"the transfer's source cannot be empty"))
				})

				Convey("Given that the destination is missing", func() {
					trans.DestFile = ""
					shouldFailWith("the destination is missing", database.NewValidationError(
						"the transfer's destination cannot be empty"))
				})

				Convey("Given that the rule id is invalid", func() {
					trans.RuleID = 1000
					shouldFailWith("the rule does not exist", database.NewValidationError(
						"the rule %d does not exist", trans.RuleID))
				})

				Convey("Given that the remote id is invalid", func() {
					trans.AgentID = 1000
					shouldFailWith("the partner does not exist", database.NewValidationError(
						"the partner %d does not exist", trans.AgentID))
				})

				Convey("Given that the account id is invalid", func() {
					trans.AccountID = 1000
					shouldFailWith("the account does not exist", database.NewValidationError(
						"the agent %d does not have an account %d", trans.AgentID,
						trans.AccountID))
				})

				Convey("Given that the account id does not belong to the agent", func() {
					remote2 := RemoteAgent{
						Name:        "remote2",
						Protocol:    dummyProto,
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:2022",
					}
					So(db.Insert(&remote2).Run(), ShouldBeNil)

					account2 := RemoteAccount{
						RemoteAgentID: remote2.ID,
						Login:         "titi",
						Password:      "password",
					}
					So(db.Insert(&account2).Run(), ShouldBeNil)

					trans.AgentID = remote.ID
					trans.AccountID = account2.ID

					shouldFailWith("the account does not exist", database.NewValidationError(
						"the agent %d does not have an account %d", trans.AgentID,
						trans.AccountID))
				})

				statusTestCases := []statusTestCase{
					{StatusPlanned, true},
					{StatusRunning, true},
					{StatusDone, false},
					{StatusError, true},
					{StatusCancelled, false},
					{"toto", false},
				}
				for _, tc := range statusTestCases {
					testTransferStatus(tc, &trans, db)
				}
			})
		})
	})
}

func TestTransferToHistory(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		remote := RemoteAgent{
			Name:        "remote",
			Protocol:    dummyProto,
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:2022",
		}
		So(db.Insert(&remote).Run(), ShouldBeNil)

		account := RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "toto",
			Password:      "password",
		}
		So(db.Insert(&account).Run(), ShouldBeNil)

		rule := Rule{
			Name:   "rule1",
			IsSend: true,
			Path:   "path",
		}
		So(db.Insert(&rule).Run(), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := Transfer{
				ID:         1,
				RuleID:     rule.ID,
				IsServer:   false,
				AgentID:    remote.ID,
				AccountID:  account.ID,
				SourceFile: "test/source/path",
				DestFile:   "test/dest/path",
				Start:      time.Now(),
				Status:     StatusDone,
				Owner:      conf.GlobalConfig.GatewayName,
			}

			Convey("When calling the `ToHistory` method", func() {
				stop := time.Now()
				var hist *TransferHistory
				So(db.Transaction(func(ses *database.Session) database.Error {
					var err database.Error
					hist, err = trans.ToHistory(ses, stop)
					return err
				}), ShouldBeNil)

				Convey("Then it should return an equivalent `TransferHistory` entry", func() {
					expected := &TransferHistory{
						ID:             trans.ID,
						Owner:          trans.Owner,
						IsServer:       false,
						IsSend:         true,
						Account:        account.Login,
						Agent:          remote.Name,
						Protocol:       remote.Protocol,
						SourceFilename: trans.SourceFile,
						DestFilename:   trans.DestFile,
						Rule:           rule.Name,
						Start:          trans.Start,
						Stop:           stop,
						Status:         trans.Status,
					}

					So(hist, ShouldResemble, expected)
				})

				type statusTestCase struct {
					status          TransferStatus
					expectedSuccess bool
				}
				statusesTestCases := []statusTestCase{
					{StatusPlanned, false},
					{StatusRunning, false},
					{StatusDone, true},
					{StatusError, false},
					{StatusCancelled, true},
					{"toto", false},
				}

				for _, tc := range statusesTestCases {
					Convey(fmt.Sprintf("Given the status is set to '%s'", tc.status), func() {
						trans.Status = tc.status

						Convey("When calling the `ToHistory` method", func() {
							var h *TransferHistory
							err := db.Transaction(func(ses *database.Session) database.Error {
								var err database.Error
								h, err = trans.ToHistory(ses, stop)
								return err
							})

							if tc.expectedSuccess {
								Convey("Then it should not return any error", func() {
									So(err, ShouldBeNil)
								})
								Convey("Then it should return a History object", func() {
									So(h, ShouldNotBeNil)
								})
							} else {
								Convey("Then it should return an error", func() {
									expectedError := fmt.Sprintf(
										"a transfer cannot be recorded in history with status '%s'",
										tc.status,
									)
									So(err, ShouldBeError, expectedError)
								})
								Convey("Then it should not return a History object", func() {
									So(h, ShouldBeNil)
								})
							}
						})
					})
				}
			})
		})
	})
}
