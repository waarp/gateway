package pipelinetest

import (
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type transData struct {
	ClientTrans *model.Transfer
	// fileInfo     map[string]interface{}
	fileContent []byte
}

func (d *clientData) checkClientTransferOK(c convey.C, data *transData,
	actual *model.HistoryEntry,
) {
	c.Convey("Then there should be a client-side history entry", func(c convey.C) {
		expected := &model.HistoryEntry{
			Identifier:       data.ClientTrans.Identifier,
			RemoteTransferID: actual.RemoteTransferID,
			Owner:            actual.Owner,
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
			TransferInfo:     data.ClientTrans.TransferInfo,
			Infos:            actual.Infos,
		}
		c.So(actual, convey.ShouldResemble, expected)
	})
}

func (d *serverData) checkServerTransferOK(c convey.C, remoteTransferID, filename string,
	progress int64, ctx *testData, actual *model.HistoryEntry, transInfo map[string]any,
) {
	c.Convey("Then there should be a server-side history entry", func(c convey.C) {
		expectedLocalPath := fs.JoinPath(ctx.Paths.GatewayHome, d.Server.RootDir,
			d.ServerRule.LocalDir, filename)

		expected := &model.HistoryEntry{
			Identifier:       actual.Identifier,
			RemoteTransferID: remoteTransferID,
			Owner:            actual.Owner,
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
			TransferInfo:     transInfo,
			Infos:            actual.Infos,
		}

		if d.ServerRule.IsSend {
			expected.SrcFilename = filename
		} else {
			expected.DestFilename = filename
		}

		c.So(*actual, convey.ShouldResemble, *expected)
	})
}
