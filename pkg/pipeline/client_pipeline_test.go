package pipeline

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestClientPipelineRun(t *testing.T) {
	content := []byte("client pipeline test file content")

	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a client push transfer", func() {
			file := "client_pipeline_push"
			So(ioutil.WriteFile(filepath.Join(conf.GlobalConfig.Paths.GatewayHome,
				ctx.send.LocalDir, file), content, 0o600), ShouldBeNil)

			trans := &model.Transfer{
				IsServer:   false,
				RuleID:     ctx.send.ID,
				AgentID:    ctx.partner.ID,
				AccountID:  ctx.remoteAccount.ID,
				LocalPath:  file,
				RemotePath: file,
				Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
				Status:     types.StatusPlanned,
			}
			So(ctx.db.Insert(trans).Run(), ShouldBeNil)

			Convey("When launching the transfer pipeline", func() {
				pip, err := NewClientPipeline(ctx.db, trans)
				So(err, ShouldBeNil)
				pip.Run()

				Convey("Then the transfer should be in the history", func() {
					exp := model.HistoryEntry{
						ID:               trans.ID,
						Owner:            conf.GlobalConfig.GatewayName,
						RemoteTransferID: trans.RemoteTransferID,
						IsServer:         false,
						IsSend:           true,
						Rule:             ctx.send.Name,
						Agent:            ctx.partner.Name,
						Account:          ctx.remoteAccount.Login,
						Protocol:         testProtocol,
						LocalPath:        trans.LocalPath,
						RemotePath:       trans.RemotePath,
						Filesize:         int64(len(content)),
						Start:            trans.Start.Local(),
						Status:           types.StatusDone,
						Step:             0,
						Progress:         uint64(len(content)),
						TaskNumber:       0,
						Error:            types.TransferError{},
					}

					var hist model.HistoryEntries
					So(ctx.db.Select(&hist).Run(), ShouldBeNil)
					So(hist, ShouldNotBeEmpty)
					exp.Stop = hist[0].Stop
					So(hist[0], ShouldResemble, exp)
				})
			})
		})
	})
}
