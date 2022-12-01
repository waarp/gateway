package backup

import (
	"bytes"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func TestImportHistory(t *testing.T) {
	//nolint:misspell //spelling mistake must be kept for compatibility reasons
	const jsonInput = `[
  {
    "id": 1,
    "remoteId": "123",
    "rule": "rule1",
    "isSend": false,
    "isServer": true,
    "requester": "acc1",
    "requested": "agent1",
    "protocol": "test_proto",
    "localFilepath": "/path/local/1",
    "remoteFilepath": "/path/remote/1",
    "filesize": 1234,
    "start": "2021-01-01T01:00:00.123456Z",
    "stop": "2021-01-01T02:00:00.123456Z",
    "status": "DONE",
    "progress": 321,
    "taskNumber": 10,
    "transferInfo": {
      "key": "val"
    }
  },
  {
    "id": 2,
    "remoteId": "567",
    "rule": "rule2",
    "isSend": true,
    "isServer": false,
    "requester": "acc2",
    "requested": "agent2",
    "protocol": "test_proto",
    "localFilepath": "/path/local/2",
    "remoteFilepath": "/path/remote/2",
    "filesize": 5678,
    "start": "2022-01-01T01:00:00.123456Z",
    "stop": "2022-01-01T02:00:00.123456Z",
    "status": "CANCELLED",
    "step": "StepData",
    "progress": 987,
    "taskNumber": 20,
    "errorCode": "TeDataTransfer",
    "errorMsg": "error in data transfer"
  }
]
`

	Convey("Given a database with a history entry", t, func(c C) {
		db := database.TestDatabase(c)

		expected1 := &model.HistoryEntry{
			ID:               1,
			Owner:            conf.GlobalConfig.GatewayName,
			RemoteTransferID: "123",
			IsSend:           false,
			IsServer:         true,
			Rule:             "rule1",
			Account:          "acc1",
			Agent:            "agent1",
			Protocol:         testProtocol,
			LocalPath:        "/path/local/1",
			RemotePath:       "/path/remote/1",
			Filesize:         1234,
			Start:            time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.UTC).Local(),
			Stop:             time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.UTC).Local(),
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Progress:         321,
			TaskNumber:       10,
			Error:            types.TransferError{},
		}
		expected2 := &model.HistoryEntry{
			ID:               2,
			Owner:            conf.GlobalConfig.GatewayName,
			RemoteTransferID: "567",
			IsSend:           true,
			IsServer:         false,
			Rule:             "rule2",
			Account:          "acc2",
			Agent:            "agent2",
			Protocol:         testProtocol,
			LocalPath:        "/path/local/2",
			RemotePath:       "/path/remote/2",
			Filesize:         5678,
			Start:            time.Date(2022, 1, 1, 1, 0, 0, 123456000, time.UTC).Local(),
			Stop:             time.Date(2022, 1, 1, 2, 0, 0, 123456000, time.UTC).Local(),
			Status:           types.StatusCancelled,
			Step:             types.StepData,
			Progress:         987,
			TaskNumber:       20,
			Error: types.TransferError{
				Code:    types.TeDataTransfer,
				Details: "error in data transfer",
			},
		}

		hist3 := &model.HistoryEntry{
			ID:               3,
			RemoteTransferID: "789",
			IsServer:         true,
			IsSend:           true,
			Rule:             "rule3",
			Account:          "acc3",
			Agent:            "agent3",
			Protocol:         testProtocol,
			LocalPath:        "/path/local/3",
			RemotePath:       "/path/remote/3",
			Filesize:         9876,
			Start:            time.Date(2020, 1, 1, 1, 0, 0, 123456000, time.Local),
			Stop:             time.Date(2020, 1, 1, 2, 0, 0, 123456000, time.Local),
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Progress:         654,
			TaskNumber:       30,
		}
		So(db.Insert(hist3).Run(), ShouldBeNil)

		Convey("When importing the history dump file", func() {
			buf := bytes.NewBufferString(jsonInput)
			So(ImportHistory(db, buf, false), ShouldBeNil)

			Convey("Then it should have imported the history entries", func() {
				var hist model.HistoryEntries
				So(db.Select(&hist).OrderBy("id", true).Run(), ShouldBeNil)
				So(hist, ShouldHaveLength, 2)
				So(hist[0], ShouldResemble, expected1)
				So(hist[1], ShouldResemble, expected2)

				info, err := hist[0].GetTransferInfo(db)
				So(err, ShouldBeNil)
				So(info["key"], ShouldResemble, "val")
			})

			Convey("Then any newly inserted transfer should not have a conflicting ID", func() {
				rule := &model.Rule{Name: "rule", IsSend: false}
				So(db.Insert(rule).Run(), ShouldBeNil)

				locAg := &model.LocalAgent{Name: "locAg", Protocol: testProtocol, Address: "1.2.3.4:5"}
				So(db.Insert(locAg).Run(), ShouldBeNil)

				locAcc := &model.LocalAccount{LocalAgentID: locAg.ID, Login: "locAcc"}
				So(db.Insert(locAcc).Run(), ShouldBeNil)

				newTrans := &model.Transfer{
					RuleID:         rule.ID,
					LocalAccountID: utils.NewNullInt64(locAcc.ID),
					LocalPath:      "/loc/path",
					RemotePath:     "/rem/path",
				}
				So(db.Insert(newTrans).Run(), ShouldBeNil)

				So(newTrans.ID, ShouldEqual, 3)
			})
		})

		Convey("When importing the history dump file with the dry flag", func() {
			buf := bytes.NewBufferString(jsonInput)
			So(ImportHistory(db, buf, true), ShouldBeNil)

			Convey("Then it should NOT have imported the history entries", func() {
				var hist model.HistoryEntries
				So(db.Select(&hist).OrderBy("id", true).Run(), ShouldBeNil)
				So(hist, ShouldHaveLength, 1)
				So(hist[0], ShouldResemble, hist3)
			})
		})
	})
}
