package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
				So(name, ShouldEqual, "transfers")
			})
		})
	})
}

func TestTransferValidate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains a valid remote agent", func() {
			remote := &RemoteAgent{
				Name:        "remote",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(remote), ShouldBeNil)

			account := &RemoteAccount{
				RemoteAgentID: remote.ID,
				Login:         "toto",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			cert := &Cert{
				OwnerType:   remote.TableName(),
				OwnerID:     remote.ID,
				Name:        "remote_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			rule := &Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "path",
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a new transfer", func() {
				trans := &Transfer{
					RuleID:       rule.ID,
					IsServer:     false,
					AgentID:      remote.ID,
					AccountID:    account.ID,
					TrueFilepath: "/filepath",
					SourceFile:   "source",
					DestFile:     "dest",
					Start:        time.Now(),
					Status:       "PLANNED",
					Owner:        database.Owner,
				}

				Convey("Given that the new transfer is valid", func() {

					Convey("When calling the 'Validate' function", func() {
						So(trans.Validate(db), ShouldBeNil)

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

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the rule ID is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the transfer's rule ID cannot be empty"))
						})
					})
				})

				Convey("Given that the remote ID is missing", func() {
					trans.AgentID = 0

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the remote ID is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the transfer's remote ID cannot be empty"))
						})
					})
				})

				Convey("Given that the account ID is missing", func() {
					trans.AccountID = 0

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the account ID is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the transfer's account ID cannot be empty"))
						})
					})
				})

				Convey("Given that the source is missing", func() {
					trans.SourceFile = ""

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the source is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the transfer's source cannot be empty"))
						})
					})
				})

				Convey("Given that the destination is missing", func() {
					trans.DestFile = ""

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the destination is missing", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the transfer's destination cannot be empty"))
						})
					})
				})

				Convey("Given that the rule id is invalid", func() {
					trans.RuleID = 1000

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the rule does not exist", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the rule %d does not exist", trans.RuleID))
						})
					})
				})

				Convey("Given that the remote id is invalid", func() {
					trans.AgentID = 1000

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the partner does not exist", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the partner %d does not exist", trans.AgentID))
						})
					})
				})

				Convey("Given that the account id is invalid", func() {
					trans.AccountID = 1000

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the account does not exist", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the agent %d does not have an account %d",
								trans.AgentID, trans.AccountID))
						})
					})
				})

				Convey("Given that the account id does not belong to the agent", func() {
					remote2 := &RemoteAgent{
						Name:        "remote2",
						Protocol:    "sftp",
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:2022",
					}
					So(db.Create(remote2), ShouldBeNil)

					account2 := &RemoteAccount{
						RemoteAgentID: remote2.ID,
						Login:         "titi",
						Password:      []byte("password"),
					}
					So(db.Create(account2), ShouldBeNil)

					trans.AgentID = remote.ID
					trans.AccountID = account2.ID

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the account does not exist", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the agent %d does not have an account %d",
								trans.AgentID, trans.AccountID))
						})
					})
				})

				Convey("Given that the remote does not have a certificate", func() {
					So(db.Delete(cert), ShouldBeNil)

					Convey("When calling the 'Validate' function", func() {
						err := trans.Validate(db)

						Convey("Then the error should say the remote does not have a certificate", func() {
							So(err, ShouldBeError, database.NewValidationError(
								"the partner is missing an SFTP host key"))
						})
					})
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
					testTransferStatus(tc, "Validate", trans, db)
				}
			})
		})
	})
}

func TestTransferToHistory(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		remote := &RemoteAgent{
			Name:        "remote",
			Protocol:    "sftp",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:2022",
		}
		So(db.Create(remote), ShouldBeNil)

		account := &RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "toto",
			Password:      []byte("password"),
		}
		So(db.Create(account), ShouldBeNil)

		rule := &Rule{
			Name:   "rule1",
			IsSend: true,
			Path:   "path",
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := &Transfer{
				ID:         1,
				RuleID:     rule.ID,
				IsServer:   false,
				AgentID:    remote.ID,
				AccountID:  account.ID,
				SourceFile: "test/source/path",
				DestFile:   "test/dest/path",
				Start:      time.Now(),
				Status:     StatusDone,
				Owner:      database.Owner,
			}

			Convey("When calling the `ToHistory` method", func() {
				stop := time.Now()
				hist, err := trans.ToHistory(db, stop)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})

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
							h, err := trans.ToHistory(db, stop)

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
								Convey("Then it should Nnot return a History object", func() {
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
