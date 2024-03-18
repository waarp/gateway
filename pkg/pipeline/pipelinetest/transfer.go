package pipelinetest

import (
	"encoding/json"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type transData struct {
	ClientTrans  *model.Transfer
	transferInfo map[string]interface{}
	// fileInfo     map[string]interface{}
	fileContent []byte
}

func (d *clientData) checkClientTransferOK(c convey.C, data *transData,
	db *database.DB, actual *model.HistoryEntry,
) {
	c.Convey("Then there should be a client-side history entry", func(c convey.C) {
		expected := &model.HistoryEntry{
			ID:               data.ClientTrans.ID,
			RemoteTransferID: actual.RemoteTransferID,
			Owner:            conf.GlobalConfig.GatewayName,
			Protocol:         d.Partner.Protocol,
			Rule:             d.ClientRule.Name,
			IsServer:         false,
			IsSend:           d.ClientRule.IsSend,
			Account:          d.RemAccount.Login,
			Agent:            d.Partner.Name,
			Start:            actual.Start,
			Stop:             actual.Stop,
			SrcFilename:      data.ClientTrans.SrcFilename,
			DestFilename:     data.ClientTrans.DestFilename,
			LocalPath:        data.ClientTrans.LocalPath,
			RemotePath:       data.ClientTrans.RemotePath,
			Filesize:         TestFileSize,
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Error:            types.TransferError{},
			Progress:         int64(len(data.fileContent)),
			TaskNumber:       0,
		}
		c.So(*actual, convey.ShouldResemble, *expected)
		checkHistoryInfo(c, db, actual.ID, data)
		/* if !d.ClientRule.IsSend {
			checkFileInfo(c, db, actual.ID, data)
		} */
	})
}

func (d *serverData) checkServerTransferOK(c convey.C, remoteTransferID, filename string,
	progress int64, ctx *testData, actual *model.HistoryEntry, data *transData,
) {
	c.Convey("Then there should be a server-side history entry", func(c convey.C) {
		expectedLocalPath := mkURL(ctx.Paths.GatewayHome, d.Server.RootDir,
			d.ServerRule.LocalDir, filename)

		expected := &model.HistoryEntry{
			ID:               actual.ID,
			RemoteTransferID: remoteTransferID,
			Owner:            conf.GlobalConfig.GatewayName,
			Protocol:         d.Server.Protocol,
			IsServer:         true,
			IsSend:           d.ServerRule.IsSend,
			Rule:             d.ServerRule.Name,
			Account:          d.LocAccount.Login,
			Agent:            d.Server.Name,
			Start:            actual.Start,
			Stop:             actual.Stop,
			LocalPath:        *expectedLocalPath,
			RemotePath:       "",
			Filesize:         TestFileSize,
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Error:            types.TransferError{},
			Progress:         progress,
			TaskNumber:       0,
		}

		if d.ServerRule.IsSend {
			expected.SrcFilename = filename
		} else {
			expected.DestFilename = filename
		}

		c.So(*actual, convey.ShouldResemble, *expected)
		checkHistoryInfo(c, ctx.DB, actual.ID, data)
		/* if !d.ServerRule.IsSend {
			checkFileInfo(c, db, actual.ID, data)
		} */
	})
}

func checkHistoryInfo(c convey.C, db *database.DB, transID int64, data *transData) {
	if data == nil {
		return
	}

	var infoList model.TransferInfoList

	c.So(db.Select(&infoList).Run(), convey.ShouldBeNil)
	c.So(db.Select(&infoList).Where("history_id=?", transID).Run(), convey.ShouldBeNil)

	actualInfo := map[string]interface{}{}

	for _, info := range infoList {
		var val interface{}

		c.So(json.Unmarshal([]byte(info.Value), &val), convey.ShouldBeNil)
		actualInfo[info.Name] = val
	}

	c.So(actualInfo, convey.ShouldResemble, data.transferInfo)
}

/*
func checkFileInfo(c convey.C, db *database.DB, transID uint64, data *transData) {
	if data == nil {
		return
	}

	var infoList model.FileInfoList
	c.So(db.Select(&infoList).Where("transfer_id=?", transID).Run(), convey.ShouldBeNil)

	actualInfo := map[string]interface{}{}

	for _, info := range infoList {
		var val interface{}
		c.So(json.Unmarshal([]byte(info.Value), &val), convey.ShouldBeNil)

		actualInfo[info.Name] = val
	}

	c.So(actualInfo, convey.ShouldResemble, data.fileInfo)
}
*/
