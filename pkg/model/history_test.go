package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	. "code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHistoryTableName(t *testing.T) {
	Convey("Given a `TransferHistory` instance", t, func() {
		hist := &TransferHistory{}

		Convey("When calling the 'TableName' method", func() {
			name := hist.TableName()

			Convey("Then it should return the name of the history table", func() {
				So(name, ShouldEqual, TableHistory)
			})
		})
	})
}

func TestHistoryBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a new history entry", func() {
			hist := &TransferHistory{
				ID:             1,
				Rule:           "rule",
				IsServer:       true,
				IsSend:         true,
				Agent:          "from",
				Account:        "to",
				SourceFilename: "test/source/path",
				DestFilename:   "test/source/path",
				Start:          time.Now(),
				Stop:           time.Now(),
				Protocol:       dummyProto,
				Status:         "DONE",
				Owner:          database.Owner,
			}

			shouldFailWith := func(errDesc string, expErr error) {
				Convey("When calling the 'BeforeWrite' function", func() {
					err := hist.BeforeWrite(db)

					Convey("Then the error should say that "+errDesc, func() {
						So(err, ShouldBeError, expErr)
					})
				})
			}

			Convey("Given that the new transfer is valid", func() {
				Convey("When calling the 'BeforeWrite' function", func() {
					err := hist.BeforeWrite(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule name is missing", func() {
				hist.Rule = ""
				shouldFailWith("the rule is missing", database.NewValidationError(
					"the transfer's rule cannot be empty"))
			})

			Convey("Given that the account is missing", func() {
				hist.Account = ""
				shouldFailWith("source is missing", database.NewValidationError(
					"the transfer's account cannot be empty"))
			})

			Convey("Given that the agent is missing", func() {
				hist.Agent = ""
				shouldFailWith("the destination is missing", database.NewValidationError(
					"the transfer's agent cannot be empty"))
			})

			Convey("Given that the filename is missing", func() {
				hist.DestFilename = ""
				shouldFailWith("the filename is missing", database.NewValidationError(
					"the transfer's destination filename cannot be empty"))
			})

			Convey("Given that the protocol is invalid", func() {
				hist.Protocol = "invalid"
				shouldFailWith("the protocol is missing", database.NewValidationError(
					"'invalid' is not a valid protocol"))
			})

			Convey("Given that the starting date is missing", func() {
				hist.Start = time.Time{}
				shouldFailWith("the start date is missing", database.NewValidationError(
					"the transfer's start date cannot be empty"))
			})

			Convey("Given that the end date is before the start date", func() {
				hist.Stop = hist.Start.AddDate(0, 0, -1)
				shouldFailWith("the end date is anterior", database.NewValidationError(
					"the transfer's end date cannot be anterior to the start date"))
			})

			statusTestCases := []statusTestCase{
				{StatusPlanned, false},
				{StatusRunning, false},
				{StatusDone, true},
				{StatusError, false},
				{StatusCancelled, true},
				{"toto", false},
			}
			for _, tc := range statusTestCases {
				testTransferStatus(tc, hist, db)
			}
		})
	})
}

//
// Test utils
//

type statusTestCase struct {
	status          TransferStatus
	expectedSuccess bool
}

func testTransferStatus(tc statusTestCase, target database.WriteHook, db *database.DB) {
	Convey(fmt.Sprintf("Given the status is set to '%s'", tc.status), func() {
		var typeName string
		if t, ok := target.(*TransferHistory); ok {
			t.Status = tc.status
			typeName = "transfer history"
		}
		if t, ok := target.(*Transfer); ok {
			t.Status = tc.status
			typeName = "transfer"
		}

		Convey("When the 'BeforeWrite' method is called", func() {
			err := target.BeforeWrite(db)

			if tc.expectedSuccess {
				Convey("Then it should not return any error", func() {
					So(err, ShouldBeNil)
				})
			} else {
				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the status is invalid", func() {
					So(err, ShouldBeError, database.NewValidationError(
						"'%s' is not a valid %s status", tc.status, typeName))
				})
			}
		})
	})
}

func TestTransferHistoryRestart(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		rule := &Rule{Name: "rule", IsSend: true}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given a client history entry", func() {
			agent := &RemoteAgent{
				Name:        "partner",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "toto",
				Password:      "password",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			history := &TransferHistory{
				ID:               1,
				Owner:            database.Owner,
				RemoteTransferID: "2",
				IsServer:         false,
				IsSend:           rule.IsSend,
				Account:          account.Login,
				Agent:            agent.Name,
				Protocol:         agent.Protocol,
				SourceFilename:   "file.src",
				DestFilename:     "file.dst",
				Rule:             rule.Name,
				Start:            time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Status:           types.StatusDone,
				Error:            TransferError{},
				Step:             types.StepNone,
				Progress:         100,
				TaskNumber:       0,
			}

			Convey("When calling the 'Restart' function", func() {
				date := time.Date(2021, 0, 0, 0, 0, 0, 0, time.Local)
				trans, err := history.Restart(db, date)
				So(err, ShouldBeNil)

				Convey("Then it should return a new transfer instance", func() {
					exp := &Transfer{
						ID:               0,
						RemoteTransferID: "2",
						RuleID:           rule.ID,
						IsServer:         false,
						AgentID:          agent.ID,
						AccountID:        account.ID,
						TrueFilepath:     "",
						SourceFile:       "file.src",
						DestFile:         "file.dst",
						Start:            date,
						Step:             types.StepNone,
						Status:           types.StatusPlanned,
						Owner:            database.Owner,
						Progress:         0,
						TaskNumber:       0,
						Error:            TransferError{},
					}
					So(trans, ShouldResemble, exp)
				})
			})
		})

		Convey("Given a server history entry", func() {
			agent := &LocalAgent{
				Name:        "server",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
				PasswordHash: hash("password"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			history := &TransferHistory{
				ID:               1,
				Owner:            database.Owner,
				RemoteTransferID: "2",
				IsServer:         true,
				IsSend:           rule.IsSend,
				Account:          account.Login,
				Agent:            agent.Name,
				Protocol:         agent.Protocol,
				SourceFilename:   "file.src",
				DestFilename:     "file.dst",
				Rule:             rule.Name,
				Start:            time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Status:           types.StatusDone,
				Error:            TransferError{},
				Step:             types.StepNone,
				Progress:         100,
				TaskNumber:       0,
			}

			Convey("When calling the 'Restart' function", func() {
				date := time.Date(2021, 0, 0, 0, 0, 0, 0, time.Local)
				trans, err := history.Restart(db, date)
				So(err, ShouldBeNil)

				Convey("Then it should return a new transfer instance", func() {
					exp := &Transfer{
						ID:               0,
						RemoteTransferID: "2",
						RuleID:           rule.ID,
						IsServer:         true,
						AgentID:          agent.ID,
						AccountID:        account.ID,
						TrueFilepath:     "",
						SourceFile:       "file.src",
						DestFile:         "file.dst",
						Start:            date,
						Step:             types.StepNone,
						Status:           types.StatusPlanned,
						Owner:            database.Owner,
						Progress:         0,
						TaskNumber:       0,
						Error:            TransferError{},
					}
					So(trans, ShouldResemble, exp)
				})
			})
		})
	})
}
