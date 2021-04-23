package pipeline

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestClientPipelineRun(t *testing.T) {
	content := []byte("client pipeline test file content")

	Convey("Given a database", t, func(c C) {
		ctx := testhelpers.InitClient(c, testhelpers.TestProtocol, nil)
		testhelpers.MakeChan(c)

		Convey("Given a client push transfer", func() {
			file := "client_pipeline_push"
			So(ioutil.WriteFile(filepath.Join(ctx.Paths.GatewayHome, ctx.Paths.DefaultOutDir,
				file), content, 0600), ShouldBeNil)

			trans := &model.Transfer{
				IsServer:   false,
				RuleID:     ctx.ClientPush.ID,
				AgentID:    ctx.Partner.ID,
				AccountID:  ctx.RemAccount.ID,
				LocalPath:  file,
				RemotePath: file,
				Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
				Status:     types.StatusPlanned,
			}
			So(ctx.DB.Insert(trans).Run(), ShouldBeNil)

			Convey("When launching the transfer pipeline", func() {
				pip, err := NewClientPipeline(ctx.DB, trans)
				So(err, ShouldBeNil)
				pip.Run()

				Convey("Then the transfer should be in the history", func() {
					exp := model.HistoryEntry{
						ID:               trans.ID,
						Owner:            database.Owner,
						RemoteTransferID: "",
						IsServer:         false,
						IsSend:           true,
						Rule:             ctx.ClientPush.Name,
						Agent:            ctx.Partner.Name,
						Account:          ctx.RemAccount.Login,
						Protocol:         "test",
						LocalPath:        trans.LocalPath,
						RemotePath:       trans.RemotePath,
						Start:            trans.Start.Local(),
						Status:           types.StatusDone,
						Step:             0,
						Progress:         uint64(len(content)),
						TaskNumber:       0,
						Error:            types.TransferError{},
					}

					var hist model.HistoryEntries
					So(ctx.DB.Select(&hist).Run(), ShouldBeNil)
					So(hist, ShouldNotBeEmpty)
					exp.Stop = hist[0].Stop
					So(hist[0], ShouldResemble, exp)
				})
			})
		})
	})
}
