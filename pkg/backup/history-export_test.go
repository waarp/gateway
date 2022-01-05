package backup

import (
	"bytes"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestExportHistory(t *testing.T) {
	Convey("Given a database with 2 history entries", t, func(c C) {
		db := database.TestDatabase(c)

		hist1 := &model.HistoryEntry{
			ID:               1,
			RemoteTransferID: "123",
			IsServer:         true,
			IsSend:           false,
			Rule:             "rule1",
			Account:          "acc1",
			Agent:            "agent1",
			Protocol:         testProtocol,
			LocalPath:        "/path/local/1",
			RemotePath:       "/path/remote/1",
			Filesize:         1234,
			Start:            time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.Local),
			Stop:             time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.Local),
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Progress:         321,
			TaskNumber:       10,
		}
		So(db.Insert(hist1).Run(), ShouldBeNil)
		So(hist1.SetTransferInfo(db, map[string]interface{}{"key": "val"}), ShouldBeNil)

		hist2 := &model.HistoryEntry{
			ID:               2,
			RemoteTransferID: "567",
			IsServer:         false,
			IsSend:           true,
			Rule:             "rule2",
			Account:          "acc2",
			Agent:            "agent2",
			Protocol:         testProtocol,
			LocalPath:        "/path/local/2",
			RemotePath:       "/path/remote/2",
			Filesize:         5678,
			Start:            time.Date(2022, 1, 1, 1, 0, 0, 123456000, time.Local),
			Stop:             time.Date(2022, 1, 1, 2, 0, 0, 123456000, time.Local),
			Status:           types.StatusCancelled,
			Step:             types.StepData,
			Progress:         987,
			TaskNumber:       20,
			Error: types.TransferError{
				Code:    types.TeDataTransfer,
				Details: "error in data transfer",
			},
		}
		So(db.Insert(hist2).Run(), ShouldBeNil)

		Convey("When exporting the history", func() {
			buf := &bytes.Buffer{}
			So(ExportHistory(db, buf, time.Time{}), ShouldBeNil)

			Convey("Then it should have written the JSON to the output", func() {
				//nolint:misspell //spelling mistake must be kept for compatibility reasons
				So(buf.String(), ShouldEqual, `[
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
    "start": "2021-01-01T01:00:00.123456+01:00",
    "stop": "2021-01-01T02:00:00.123456+01:00",
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
    "start": "2022-01-01T01:00:00.123456+01:00",
    "stop": "2022-01-01T02:00:00.123456+01:00",
    "status": "CANCELLED",
    "step": "StepData",
    "progress": 987,
    "taskNumber": 20,
    "errorCode": "TeDataTransfer",
    "errorMsg": "error in data transfer"
  }
]
`)
			})

			Convey("Then the database entries should be unchanged", func() {
				var hist model.HistoryEntries
				So(db.Select(&hist).OrderBy("id", true).Run(), ShouldBeNil)
				So(hist, ShouldHaveLength, 2)
				So(hist[0], ShouldResemble, hist1)
				So(hist[1], ShouldResemble, hist2)
			})
		})

		Convey("When exporting the history with a time", func() {
			buf := &bytes.Buffer{}
			So(ExportHistory(db, buf, time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)), ShouldBeNil)

			Convey("Then it should have written the JSON to the output", func() {
				So(buf.String(), ShouldEqual, `[
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
    "start": "2021-01-01T01:00:00.123456+01:00",
    "stop": "2021-01-01T02:00:00.123456+01:00",
    "status": "DONE",
    "progress": 321,
    "taskNumber": 10,
    "transferInfo": {
      "key": "val"
    }
  }
]
`)
			})
		})
	})
}
