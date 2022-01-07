package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
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
		db := database.TestDatabase(c)

		Convey("Given the database contains a valid remote agent", func() {
			server := LocalAgent{
				Name:     "remote",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(&server).Run(), ShouldBeNil)

			account := LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
				PasswordHash: hash("sesame"),
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
					RemoteTransferID: "1",
					RuleID:           rule.ID,
					LocalAccountID:   utils.NewNullInt64(account.ID),
					LocalPath:        "/local/path",
					RemotePath:       "/remote/path",
					Start:            time.Now(),
					Status:           types.StatusPlanned,
					Owner:            conf.GlobalConfig.GatewayName,
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

						Convey("Then the remote ID should have been initialized", func() {
							So(trans.RemoteTransferID, ShouldNotBeBlank)
						})

						Convey("Then the transfer status should be 'planned'", func() {
							So(trans.Status, ShouldEqual, "PLANNED")
						})

						Convey("Then the transfer owner should be 'test_gateway'", func() {
							So(trans.Owner, ShouldEqual, "test_gateway")
						})

						Convey("Then the local path should be in the OS's format", func() {
							So(trans.LocalPath, ShouldEqual, utils.ToOSPath("/local/path"))
						})
					})
				})

				Convey("Given that the rule ID is missing", func() {
					trans.RuleID = 0
					shouldFailWith("the rule ID is missing", database.NewValidationError(
						"the transfer's rule ID cannot be empty"))
				})

				Convey("Given that the account ID is missing", func() {
					trans.LocalAccountID = sql.NullInt64{}
					shouldFailWith("the remote ID is missing", database.NewValidationError(
						"the transfer is missing an account ID"))
				})

				Convey("Given that the transfer has both account IDs", func() {
					trans.RemoteAccountID = utils.NewNullInt64(1)
					shouldFailWith("the account ID is missing", database.NewValidationError(
						"the transfer cannot have both a local and remote account ID"))
				})

				Convey("Given that the local filepath is missing", func() {
					trans.LocalPath = ""
					shouldFailWith("the local filepath is missing", database.NewValidationError(
						"the local filepath is missing"))
				})

				Convey("Given that the rule id is invalid", func() {
					trans.RuleID = 1000
					shouldFailWith("the rule does not exist", database.NewValidationError(
						"the rule %d does not exist", trans.RuleID))
				})

				Convey("Given that the account id is invalid", func() {
					trans.LocalAccountID = utils.NewNullInt64(1000)
					shouldFailWith("the local account does not exist", database.NewValidationError(
						"the local account %d does not exist", trans.LocalAccountID.Int64))
				})

				Convey("Given that an transfer with the same remoteID already exist", func() {
					t2 := &Transfer{
						Owner:            conf.GlobalConfig.GatewayName,
						RemoteTransferID: trans.RemoteTransferID,
						RuleID:           rule.ID,
						LocalAccountID:   utils.NewNullInt64(account.ID),
						LocalPath:        "/local/path",
						RemotePath:       "/remote/path",
						Filesize:         -1,
						Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
						Status:           types.StatusRunning,
					}
					So(db.Insert(t2).Run(), ShouldBeNil)

					shouldFailWith("the remoteID is already taken", database.NewValidationError(
						"a transfer from the same account with the same remote ID already exists"))
				})

				Convey("Given that an history entry with the same remoteID already exist", func() {
					t2 := &HistoryEntry{
						ID:               10,
						Owner:            conf.GlobalConfig.GatewayName,
						RemoteTransferID: trans.RemoteTransferID,
						Protocol:         testProtocol,
						IsServer:         true,
						Rule:             rule.Name,
						Agent:            server.Name,
						Account:          account.Login,
						LocalPath:        "/local/path",
						RemotePath:       "/remote/path",
						Filesize:         100,
						Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
						Stop:             time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC),
						Status:           types.StatusDone,
					}
					So(db.Insert(t2).Run(), ShouldBeNil)

					shouldFailWith("the remoteID is already taken", database.NewValidationError(
						"a transfer from the same account with the same remote ID already exists"))
				})

				statusTestCases := []statusTestCase{
					{types.StatusPlanned, true},
					{types.StatusRunning, true},
					{types.StatusDone, false},
					{types.StatusError, true},
					{types.StatusCancelled, false},
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
		logger := testhelpers.TestLogger(c, "test_to_history")
		db := database.TestDatabase(c)

		remote := RemoteAgent{
			Name:        "remote",
			Protocol:    testProtocol,
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:2022",
		}
		So(db.Insert(&remote).Run(), ShouldBeNil)

		account := RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "toto",
			Password:      "sesame",
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
				ID:              1,
				RuleID:          rule.ID,
				RemoteAccountID: utils.NewNullInt64(account.ID),
				LocalPath:       "/test/local/path",
				RemotePath:      "/test/remote/path",
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
				Status:          types.StatusPlanned,
				Owner:           conf.GlobalConfig.GatewayName,
			}
			So(db.Insert(&trans).Run(), ShouldBeNil)

			Convey("When calling the `MoveToHistory` method", func() {
				trans.Status = types.StatusDone
				end := time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local)
				So(trans.MoveToHistory(db, logger, end), ShouldBeNil)

				Convey("Then it should have inserted an equivalent `HistoryEntry` entry", func() {
					var hist HistoryEntry
					So(db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)

					expected := HistoryEntry{
						ID:               trans.ID,
						RemoteTransferID: trans.RemoteTransferID,
						Owner:            trans.Owner,
						IsServer:         false,
						IsSend:           true,
						Account:          account.Login,
						Agent:            remote.Name,
						Protocol:         remote.Protocol,
						LocalPath:        trans.LocalPath,
						RemotePath:       trans.RemotePath,
						Rule:             rule.Name,
						Start:            trans.Start,
						Stop:             end,
						Status:           trans.Status,
					}

					So(hist, ShouldResemble, expected)
				})

				Convey("Then it should have removed the old transfer entry", func() {
					var results Transfers
					So(db.Select(&results).Run(), ShouldBeNil)
					So(results, ShouldBeEmpty)
				})
			})

			type statusTestCase struct {
				status          types.TransferStatus
				expectedSuccess bool
			}
			statusesTestCases := []statusTestCase{
				{types.StatusPlanned, false},
				{types.StatusRunning, false},
				{types.StatusDone, true},
				{types.StatusError, false},
				{types.StatusCancelled, true},
				{"toto", false},
			}

			for _, tc := range statusesTestCases {
				Convey(fmt.Sprintf("Given the status is set to '%s'", tc.status), func() {
					trans.Status = tc.status

					Convey("When calling the `MoveToHistory` method", func() {
						err := trans.MoveToHistory(db, logger, time.Now())
						var hist HistoryEntries
						So(db.Select(&hist).Run(), ShouldBeNil)

						if tc.expectedSuccess {
							Convey("Then it should not return any error", func() {
								So(err, ShouldBeNil)
							})
							Convey("Then it should have inserted a HistoryEntry object", func() {
								So(hist, ShouldNotBeEmpty)
							})
						} else {
							Convey("Then it should return an error", func() {
								expectedError := fmt.Sprintf(
									"a transfer cannot be recorded in history with status '%s'",
									tc.status,
								)
								So(err, ShouldBeError, expectedError)
							})
							Convey("Then it should NOT have inserted a HistoryEntry object", func() {
								So(hist, ShouldBeEmpty)
							})
						}
					})
				})
			}
		})
	})
}
