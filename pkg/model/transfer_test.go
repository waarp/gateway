package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
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
			server := LocalAgent{
				Name:        "remote",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
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
					IsServer:         true,
					AgentID:          server.ID,
					AccountID:        account.ID,
					LocalPath:        "/local/path",
					RemotePath:       "/remote/path",
					Start:            time.Now(),
					Status:           StatusPlanned,
					Owner:            database.Owner,
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

				Convey("Given that the remote id is invalid", func() {
					trans.AgentID = 1000
					shouldFailWith("the server does not exist", database.NewValidationError(
						"the server %d does not exist", trans.AgentID))
				})

				Convey("Given that the account id is invalid", func() {
					trans.AccountID = 1000
					shouldFailWith("the account does not exist", database.NewValidationError(
						"the server %d does not have an account %d", trans.AgentID,
						trans.AccountID))
				})

				Convey("Given that an transfer with the same remoteID already exist", func() {
					t2 := &Transfer{
						Owner:            database.Owner,
						RemoteTransferID: trans.RemoteTransferID,
						IsServer:         true,
						RuleID:           rule.ID,
						AgentID:          server.ID,
						AccountID:        account.ID,
						LocalPath:        "/local/path",
						RemotePath:       "/remote/path",
						Filesize:         -1,
						Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
						Status:           StatusRunning,
					}
					So(db.Insert(t2).Run(), ShouldBeNil)

					shouldFailWith("the remoteID is already taken", database.NewValidationError(
						"a transfer from the same account with the same remote ID already exists"))
				})

				Convey("Given that an history entry with the same remoteID already exist", func() {
					t2 := &HistoryEntry{
						ID:               10,
						Owner:            database.Owner,
						RemoteTransferID: trans.RemoteTransferID,
						Protocol:         testhelpers.TestProtocol,
						IsServer:         true,
						Rule:             rule.Name,
						Agent:            server.Name,
						Account:          account.Login,
						LocalPath:        "/local/path",
						RemotePath:       "/remote/path",
						Filesize:         100,
						Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
						Stop:             time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC),
						Status:           StatusDone,
					}
					So(db.Insert(t2).Run(), ShouldBeNil)

					shouldFailWith("the remoteID is already taken", database.NewValidationError(
						"a transfer from the same account with the same remote ID already exists"))
				})

				Convey("Given that the account id does not belong to the agent", func() {
					server2 := LocalAgent{
						Name:        "remote2",
						Protocol:    dummyProto,
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:2022",
					}
					So(db.Insert(&server2).Run(), ShouldBeNil)

					account2 := LocalAccount{
						LocalAgentID: server2.ID,
						Login:        "titi",
						PasswordHash: hash("sesame"),
					}
					So(db.Insert(&account2).Run(), ShouldBeNil)

					trans.AgentID = server.ID
					trans.AccountID = account2.ID

					shouldFailWith("the account does not exist", database.NewValidationError(
						"the server %d does not have an account %d", trans.AgentID,
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
	logger := log.NewLogger("test_to_history")

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
				ID:         1,
				RuleID:     rule.ID,
				IsServer:   false,
				AgentID:    remote.ID,
				AccountID:  account.ID,
				LocalPath:  "/test/local/path",
				RemotePath: "/test/remote/path",
				Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
				Status:     StatusPlanned,
				Owner:      database.Owner,
			}
			So(db.Insert(&trans).Run(), ShouldBeNil)

			Convey("When calling the `ToHistory` method", func() {
				trans.Status = StatusDone
				end := time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local)
				So(trans.ToHistory(db, logger, end), ShouldBeNil)

				Convey("Then it should have inserted an equivalent `HistoryEntry` entry", func() {
					var hist HistoryEntry
					So(db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)

					expected := HistoryEntry{
						ID:         trans.ID,
						Owner:      trans.Owner,
						IsServer:   false,
						IsSend:     true,
						Account:    account.Login,
						Agent:      remote.Name,
						Protocol:   remote.Protocol,
						LocalPath:  trans.LocalPath,
						RemotePath: trans.RemotePath,
						Rule:       rule.Name,
						Start:      trans.Start,
						Stop:       end,
						Status:     trans.Status,
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
						err := trans.ToHistory(db, logger, time.Now())
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
