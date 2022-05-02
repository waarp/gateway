package pipelinetest

import (
	"path/filepath"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type transData struct {
	ClientTrans *model.Transfer
	fileContent []byte
	// locFileName string
	remFileName string
}

func (d *clientData) checkClientTransferOK(c convey.C, t *transData,
	actual *model.HistoryEntry,
) {
	c.Convey("Then there should be a client-side history entry", func(c convey.C) {
		expected := &model.HistoryEntry{
			ID:               t.ClientTrans.ID,
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
			LocalPath:        t.ClientTrans.LocalPath,
			RemotePath:       t.ClientTrans.RemotePath,
			Filesize:         TestFileSize,
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Error:            types.TransferError{},
			Progress:         uint64(len(t.fileContent)),
			TaskNumber:       0,
		}
		c.So(*actual, convey.ShouldResemble, *expected)
	})
}

func (d *serverData) checkServerTransferOK(c convey.C, remoteTransferID, filename string,
	progress uint64, _ *database.DB, actual *model.HistoryEntry,
) {
	c.Convey("Then there should be a server-side history entry", func(c convey.C) {
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
			LocalPath:        filepath.Join(d.Server.RootDir, d.ServerRule.LocalDir, filename),
			RemotePath:       "/" + filepath.Base(filename),
			Filesize:         TestFileSize,
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Error:            types.TransferError{},
			Progress:         progress,
			TaskNumber:       0,
		}

		c.So(*actual, convey.ShouldResemble, *expected)
	})
}
