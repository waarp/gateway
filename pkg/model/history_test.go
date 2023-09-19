package model

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestHistoryTableName(t *testing.T) {
	Convey("Given a `HistoryEntry` instance", t, func() {
		hist := &HistoryEntry{}

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
		db := database.TestDatabase(c)

		Convey("Given a new history entry", func() {
			hist := &HistoryEntry{
				ID:               1,
				RemoteTransferID: "12345",
				Rule:             "rule",
				IsServer:         true,
				IsSend:           true,
				Agent:            "from",
				Account:          "to",
				SrcFilename:      "file",
				LocalPath:        mkURL(testLocalPath),
				RemotePath:       "test/remote/file",
				Start:            time.Now(),
				Stop:             time.Now(),
				Protocol:         testProtocol,
				Status:           "DONE",
				Owner:            conf.GlobalConfig.GatewayName,
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

			Convey("Given that the local path is missing", func() {
				hist.LocalPath = types.URL{}

				shouldFailWith("the local filename is missing", database.NewValidationError(
					"the local filepath cannot be empty"))
			})

			Convey("Given that the remote path is missing", func() {
				hist.IsServer = false
				hist.RemotePath = ""

				shouldFailWith("the remote filename is missing", database.NewValidationError(
					"the remote filepath cannot be empty"))
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
				{types.StatusPlanned, false},
				{types.StatusRunning, false},
				{types.StatusDone, true},
				{types.StatusError, false},
				{types.StatusCancelled, true},
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
	status          types.TransferStatus
	expectedSuccess bool
}

func testTransferStatus(tc statusTestCase, target database.WriteHook, db *database.DB) {
	Convey(fmt.Sprintf("Given the status is set to '%s'", tc.status), func() {
		var typeName string

		if t, ok := target.(*HistoryEntry); ok {
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
		db := database.TestDatabase(c)

		rule := &Rule{Name: "rule", IsSend: true}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given a client history entry", func() {
			cli := Client{Name: "client", Protocol: testProtocol}
			So(db.Insert(&cli).Run(), ShouldBeNil)

			agent := &RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:1",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "toto",
				Password:      "sesame",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			history := &HistoryEntry{
				ID:               1,
				Owner:            conf.GlobalConfig.GatewayName,
				RemoteTransferID: "12345",
				IsServer:         false,
				IsSend:           rule.IsSend,
				Client:           cli.Name,
				Account:          account.Login,
				Agent:            agent.Name,
				Protocol:         agent.Protocol,
				SrcFilename:      "file",
				LocalPath:        mkURL("file:/loc/file"),
				RemotePath:       "/rem/file",
				Rule:             rule.Name,
				Start:            time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Status:           types.StatusDone,
				Error:            types.TransferError{},
				Step:             types.StepNone,
				Progress:         100,
				TaskNumber:       0,
			}

			Convey("When calling the 'Restart' function", func() {
				date := time.Date(2021, 0, 0, 0, 0, 0, 0, time.Local)
				trans, err := history.Restart(db, date)
				So(err, ShouldBeNil)

				Convey("Then it should return a new transfer instance", func() {
					So(trans, ShouldResemble, &Transfer{
						ID:               0,
						RemoteTransferID: "",
						RuleID:           rule.ID,
						ClientID:         utils.NewNullInt64(cli.ID),
						RemoteAccountID:  utils.NewNullInt64(account.ID),
						SrcFilename:      history.SrcFilename,
						Start:            date,
						Step:             types.StepNone,
						Status:           types.StatusPlanned,
						Owner:            conf.GlobalConfig.GatewayName,
						Progress:         0,
						TaskNumber:       0,
						Error:            types.TransferError{},
					})
				})
			})
		})

		Convey("Given a server history entry", func() {
			agent := &LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:1",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account := &LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
				PasswordHash: hash("tata"),
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			history := &HistoryEntry{
				ID:               1,
				Owner:            conf.GlobalConfig.GatewayName,
				RemoteTransferID: "2",
				IsServer:         true,
				IsSend:           rule.IsSend,
				Account:          account.Login,
				Agent:            agent.Name,
				Protocol:         agent.Protocol,
				SrcFilename:      "file",
				LocalPath:        mkURL("file:/local/file"),
				RemotePath:       "/remote/file",
				Rule:             rule.Name,
				Start:            time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2020, 0, 0, 0, 0, 0, 0, time.Local),
				Status:           types.StatusDone,
				Error:            types.TransferError{},
				Step:             types.StepNone,
				Progress:         100,
				TaskNumber:       0,
			}

			Convey("When calling the 'Restart' function", func() {
				date := time.Date(2021, 0, 0, 0, 0, 0, 0, time.Local)
				trans, err := history.Restart(db, date)
				So(err, ShouldBeNil)

				Convey("Then it should return a new transfer instance", func() {
					So(trans, ShouldResemble, &Transfer{
						ID:               0,
						RemoteTransferID: "",
						RuleID:           rule.ID,
						LocalAccountID:   utils.NewNullInt64(account.ID),
						SrcFilename:      history.SrcFilename,
						Start:            date,
						Step:             types.StepNone,
						Status:           types.StatusPlanned,
						Owner:            conf.GlobalConfig.GatewayName,
						Progress:         0,
						TaskNumber:       0,
						Error:            types.TransferError{},
					})
				})
			})
		})
	})
}
