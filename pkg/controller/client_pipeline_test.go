package controller

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestClientPipelineRun(t *testing.T) {
	content := []byte("client pipeline test file content")

	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a client push transfer", func() {
			filename := "client_pipeline_push"
			filePath := mkURL(conf.GlobalConfig.Paths.GatewayHome,
				ctx.send.LocalDir, filename)
			So(fs.WriteFullFile(ctx.fs, filePath, content), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          ctx.send.ID,
				ClientID:        utils.NewNullInt64(ctx.client.ID),
				RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
				SrcFilename:     filename,
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
				Status:          types.StatusPlanned,
			}
			So(ctx.db.Insert(trans).Run(), ShouldBeNil)

			Convey("When launching the transfer pipeline", func() {
				pip, err := NewClientPipeline(ctx.db, trans)
				So(err, ShouldBeNil)
				pip.Run()

				Convey("Then the transfer should be in the history", func() {
					var hist model.HistoryEntries

					So(ctx.db.Select(&hist).Run(), ShouldBeNil)
					So(hist, ShouldNotBeEmpty)
					So(hist[0], ShouldResemble, &model.HistoryEntry{
						ID:               trans.ID,
						Owner:            conf.GlobalConfig.GatewayName,
						RemoteTransferID: trans.RemoteTransferID,
						IsServer:         false,
						IsSend:           true,
						Rule:             ctx.send.Name,
						Agent:            ctx.partner.Name,
						Account:          ctx.remoteAccount.Login,
						Client:           ctx.client.Name,
						Protocol:         testProtocol,
						SrcFilename:      trans.SrcFilename,
						LocalPath:        trans.LocalPath,
						RemotePath:       trans.RemotePath,
						Filesize:         int64(len(content)),
						Start:            trans.Start.Local(),
						Stop:             hist[0].Stop,
						Status:           types.StatusDone,
						Step:             0,
						Progress:         trans.Progress,
						TaskNumber:       0,
					})
				})
			})
		})
	})
}
