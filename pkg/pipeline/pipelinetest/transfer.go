package pipelinetest

import (
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/exp/maps"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
			Protocol:         d.Client.Protocol,
			Client:           d.Client.Name,
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
			Progress:         int64(len(data.fileContent)),
		}
		c.So(*actual, convey.ShouldResemble, *expected)
		checkHistoryInfo(c, db, actual, data)
		/* if !d.ClientRule.IsSend {
			checkFileInfo(c, db, actual.ID, data)
		} */
	})
}

func (d *serverData) checkServerTransferOK(c convey.C, remoteTransferID, filename string,
	progress int64, ctx *testData, actual *model.HistoryEntry, data *transData,
) {
	c.Convey("Then there should be a server-side history entry", func(c convey.C) {
		expectedLocalPath := mkPath(ctx.Paths.GatewayHome, d.Server.RootDir,
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
			LocalPath:        expectedLocalPath,
			RemotePath:       "",
			Filesize:         TestFileSize,
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Progress:         progress,
		}

		if d.ServerRule.IsSend {
			expected.SrcFilename = filename
		} else {
			expected.DestFilename = filename
		}

		c.So(*actual, convey.ShouldResemble, *expected)
		checkHistoryInfo(c, ctx.DB, actual, data)
		/* if !d.ServerRule.IsSend {
			checkFileInfo(c, db, actual.ID, data)
		} */
	})
}

func checkHistoryInfo(c convey.C, db *database.DB, hist *model.HistoryEntry, data *transData) {
	if data == nil || !Protocols[hist.Protocol].TransferInfo {
		return
	}

	actualInfo, err := hist.GetTransferInfo(db)
	c.So(err, convey.ShouldBeNil)

	expectedData := maps.Clone(data.transferInfo)

	var idErr error
	expectedData[model.FollowID], idErr = hist.TransferID()
	c.So(idErr, convey.ShouldBeNil)

	c.So(actualInfo, testhelpers.ShouldEqualJSON, expectedData)
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

		actualInfo[info.String] = val
	}

	c.So(actualInfo, convey.ShouldResemble, data.fileInfo)
}
*/
