package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func TestClientPipelineRun(t *testing.T) {
	content := []byte("client pipeline test file content")

	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a client push transfer", func() {
			file := "client_pipeline_push"
			So(os.WriteFile(filepath.Join(conf.GlobalConfig.Paths.GatewayHome,
				ctx.send.LocalDir, file), content, 0o600), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          ctx.send.ID,
				RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
				SrcFilename:     file,
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
						Protocol:         testProtocol,
						SrcFilename:      trans.SrcFilename,
						LocalPath:        trans.LocalPath,
						RemotePath:       trans.RemotePath,
						Filesize:         int64(len(content)),
						Start:            trans.Start.Local(),
						Stop:             hist[0].Stop,
						Status:           types.StatusDone,
						Step:             0,
						Progress:         int64(len(content)),
						TaskNumber:       0,
						Error:            types.TransferError{},
					})
				})
			})
		})
	})
}
