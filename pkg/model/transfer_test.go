package model

import (
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
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

func TestTransferBeforeInsert(t *testing.T) {
	Convey("Given a `Transfer` instance", t, func() {
		trans := &Transfer{}

		Convey("When calling the `BeforeInsert` method", func() {
			err := trans.BeforeInsert(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the transfer status should be 'planned'", func() {
				So(trans.Status, ShouldEqual, "PLANNED")
			})

			Convey("Then the transfer owner should be 'test_gateway'", func() {
				So(trans.Owner, ShouldEqual, "test_gateway")
			})
		})
	})
}

func TestTransferValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains a valid remote agent", func() {
			remote := &RemoteAgent{
				Name:        "test remote",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(remote), ShouldBeNil)

			account := &RemoteAccount{
				RemoteAgentID: remote.ID,
				Login:         "test account",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			cert := &Cert{
				OwnerType:   remote.TableName(),
				OwnerID:     remote.ID,
				Name:        "test cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			rule := &Rule{
				Name:   "test rule",
				IsSend: true,
			}
			So(db.Create(rule), ShouldBeNil)

			Convey("Given a new transfer", func() {
				trans := &Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    remote.ID,
					AccountID:  account.ID,
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Now(),
					Status:     "PLANNED",
					Owner:      database.Owner,
				}

				Convey("Given that the new transfer is valid", func() {

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the id was entered", func() {
					trans.ID = 1

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the id cannot be entered", func() {
							So(err.Error(), ShouldEqual, "The transfer's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the owner is missing", func() {
					trans.Owner = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the owner is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's owner cannot "+
								"be empty")
						})
					})
				})

				Convey("Given that the rule ID is missing", func() {
					trans.RuleID = 0

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the rule ID is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's rule ID "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the remote ID is missing", func() {
					trans.AgentID = 0

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the remote ID is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's remote ID "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the account ID is missing", func() {
					trans.AccountID = 0

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the account ID is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's account ID "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the source is missing", func() {
					trans.SourcePath = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the source is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's source "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the destination is missing", func() {
					trans.DestPath = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the destination is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's destination "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the date is missing", func() {
					trans.Start = time.Time{}

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the date is missing", func() {
							So(err.Error(), ShouldEqual, "The transfer's starting "+
								"date cannot be empty")
						})
					})
				})

				Convey("Given that the status is NOT 'planned'", func() {
					trans.Status = StatusDone

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the status must be planned", func() {
							So(err.Error(), ShouldEqual, "The transfer's status "+
								"must be 'planned' or 'pre-tasks'")
						})
					})
				})

				Convey("Given that the rule id is invalid", func() {
					trans.RuleID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the rule does not exist", func() {
							So(err.Error(), ShouldEqual, "The rule 1000 does not exist")
						})
					})
				})

				Convey("Given that the remote id is invalid", func() {
					trans.AgentID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the partner does not exist", func() {
							So(err.Error(), ShouldEqual, "The partner 1000 does not exist")
						})
					})
				})

				Convey("Given that the account id is invalid", func() {
					trans.AccountID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the account does not exist", func() {
							So(err.Error(), ShouldEqual, "The agent 1 does not have an account 1000")
						})
					})
				})

				Convey("Given that the account id does not belong to the agent", func() {
					remote2 := &RemoteAgent{
						Name:        "test remote 2",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
					}
					So(db.Create(remote2), ShouldBeNil)

					account2 := &RemoteAccount{
						RemoteAgentID: remote2.ID,
						Login:         "test account 2",
						Password:      []byte("password"),
					}
					So(db.Create(account2), ShouldBeNil)

					trans.AgentID = remote.ID
					trans.AccountID = account2.ID

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the account does not exist", func() {
							So(err.Error(), ShouldEqual, "The agent 1 does not have an account 2")
						})
					})
				})

				Convey("Given that the remote does not have a certificate", func() {
					So(db.Delete(cert), ShouldBeNil)

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = trans.ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say the remote does not have a certificate", func() {
							So(err.Error(), ShouldEqual, "No certificate found for agent 1")
						})
					})
				})
			})
		})
	})
}

func TestTransferValidateUpdate(t *testing.T) {
	Convey("Given a `Transfer` instance", t, func() {
		trans := &Transfer{
			Status:   StatusTransfer,
			Start:    time.Now(),
			IsServer: false,
		}

		Convey("Given that the entry is valid", func() {

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the entry changes the ID", func() {
			trans.ID = 1

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the ID cannot be entered", func() {
					So(err.Error(), ShouldEqual, "The transfer's ID cannot be "+
						"entered manually")
				})
			})
		})

		Convey("Given that the entry changes the rule ID", func() {
			trans.RuleID = 1

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the rule cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's rule cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the remote ID", func() {
			trans.AgentID = 1

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the partner cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's partner cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the account ID", func() {
			trans.AccountID = 1

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the account cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's account cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the owner", func() {
			trans.Owner = "owner"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the owner cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's owner cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the source", func() {
			trans.SourcePath = "source"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the source cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's source cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the destination", func() {
			trans.DestPath = "dest"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := trans.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the destination cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's destination cannot be "+
						"changed")
				})
			})
		})

		statusesTestCases := []statusTestCase{
			{StatusPlanned, true},
			{StatusTransfer, true},
			{StatusDone, false},
			{StatusError, false},
			{"toto", false},
		}

		for _, tc := range statusesTestCases {
			testHistoryStatus(tc, "ValidateUpdate", trans, nil)
		}
	})
}

func TestTransferToHistory(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		remote := &RemoteAgent{
			Name:        "test remote",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		So(db.Create(remote), ShouldBeNil)

		account := &RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      []byte("test password"),
		}
		So(db.Create(account), ShouldBeNil)

		rule := &Rule{
			Name:   "test rule",
			IsSend: true,
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := &Transfer{
				ID:         1,
				RuleID:     rule.ID,
				IsServer:   false,
				AgentID:    remote.ID,
				AccountID:  account.ID,
				SourcePath: "test/source/path",
				DestPath:   "test/dest/path",
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
						Remote:         remote.Name,
						Protocol:       remote.Protocol,
						SourceFilename: trans.SourcePath,
						DestFilename:   trans.DestPath,
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
					{StatusTransfer, false},
					{StatusDone, true},
					{StatusError, true},
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
