package model

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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

		Convey("Given the database contains a valid local agent", func() {
			server := LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(&server).Run(), ShouldBeNil)

			account := LocalAccount{
				LocalAgentID: server.ID,
				Login:        "toto",
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
					SrcFilename:      "file",
					LocalPath:        localPath(testLocalPath),
					RemotePath:       "/remote/file",
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

						Convey("Then the transfer status should be 'planned'", func() {
							So(trans.Status, ShouldEqual, types.StatusPlanned)
						})

						Convey("Then the transfer owner should be the gateway name", func() {
							So(trans.Owner, ShouldEqual, conf.GlobalConfig.GatewayName)
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

					shouldFailWith("cannot have both a local & remote account ID", database.NewValidationError(
						"the transfer cannot have both a local and remote account ID"))
				})

				Convey("Given that the filename is missing", func() {
					trans.SrcFilename = ""

					shouldFailWith("the source file is missing", database.NewValidationError(
						"the source file is missing"))
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
						SrcFilename:      "file",
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
						IsSend:           rule.IsSend,
						Rule:             rule.Name,
						Agent:            server.Name,
						Account:          account.Login,
						SrcFilename:      "file",
						LocalPath:        localPath("/local/file"),
						RemotePath:       "remote/file",
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

		Convey("Given the database contains a valid remote agent", func() {
			partner := RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(&partner).Run(), ShouldBeNil)

			account := RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
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
					RemoteTransferID: "2",
					RuleID:           rule.ID,
					RemoteAccountID:  utils.NewNullInt64(account.ID),
					SrcFilename:      "file",
					LocalPath:        localPath(testLocalPath),
					RemotePath:       "/remote/file",
					Start:            time.Now(),
					Status:           types.StatusPlanned,
					Owner:            conf.GlobalConfig.GatewayName,
				}

				Convey("Given that no client was specified", func() {
					Convey("Given that no client exists", func() {
						SoMsg("Then it should not return any error",
							trans.BeforeWrite(db), ShouldBeNil)

						Convey("Then it should have created a new default client for the transfer", func() {
							So(trans.ClientID.Valid, ShouldBeTrue)

							var client Client
							So(db.Get(&client, "id=?", trans.ClientID.Int64).Run(), ShouldBeNil)
							So(client, ShouldResemble, Client{
								Owner:       conf.GlobalConfig.GatewayName,
								ID:          trans.ClientID.Int64,
								Name:        testProtocol,
								Protocol:    testProtocol,
								ProtoConfig: map[string]any{},
							})
						})
					})

					Convey("Given that a single client exists", func() {
						client := Client{Name: "existing", Protocol: testProtocol}
						So(db.Insert(&client).Run(), ShouldBeNil)

						SoMsg("Then it should not return any error",
							trans.BeforeWrite(db), ShouldBeNil)

						Convey("Then it should use the existing client", func() {
							So(trans.ClientID.Valid, ShouldBeTrue)
							So(trans.ClientID.Int64, ShouldEqual, client.ID)
						})
					})

					Convey("Given that multiple clients exists", func() {
						client1 := Client{Name: "existing1", Protocol: testProtocol}
						So(db.Insert(&client1).Run(), ShouldBeNil)
						client2 := Client{Name: "existing2", Protocol: testProtocol}
						So(db.Insert(&client2).Run(), ShouldBeNil)

						SoMsg("Then it should return an error",
							trans.BeforeWrite(db), ShouldBeError,
							"the transfer is missing a client ID")
					})
				})

				Convey("Given that a client was specified", func() {
					client := Client{Name: "existing", Protocol: testProtocol}
					So(db.Insert(&client).Run(), ShouldBeNil)

					trans.ClientID = utils.NewNullInt64(client.ID)

					SoMsg("Then it should not return any error",
						trans.BeforeWrite(db), ShouldBeNil)
				})
			})
		})
	})
}

func TestTransferToHistory(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_to_history")
		db := database.TestDatabase(c)

		cli := Client{Name: "client", Protocol: testProtocol}
		So(db.Insert(&cli).Run(), ShouldBeNil)

		remote := RemoteAgent{
			Name: "remote", Protocol: cli.Protocol,
			Address: types.Addr("localhost", 2022),
		}
		So(db.Insert(&remote).Run(), ShouldBeNil)

		account := RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "toto",
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
				ClientID:        utils.NewNullInt64(cli.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file",
				LocalPath:       localPath(testLocalPath),
				RemotePath:      "/test/remote/file",
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
					hist := HistoryEntry{}
					So(db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)

					expected := HistoryEntry{
						ID:               trans.ID,
						RemoteTransferID: trans.RemoteTransferID,
						Owner:            trans.Owner,
						IsServer:         false,
						IsSend:           true,
						Account:          account.Login,
						Agent:            remote.Name,
						Client:           cli.Name,
						Protocol:         cli.Protocol,
						SrcFilename:      trans.SrcFilename,
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
				Convey(fmt.Sprintf("Given the status is set to %q", tc.status), func() {
					trans.Status = tc.status

					Convey("When calling the `MoveToHistory` method", func() {
						err := trans.MoveToHistory(db, logger, time.Now())

						hist := HistoryEntries{}
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
								expectedError := fmt.Sprintf("failed to move transfer to history: "+
									"a transfer cannot be recorded in history with status %q",
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
