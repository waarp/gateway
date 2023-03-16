package r66

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a r66 service with an indirect address", t, func(c C) {
		conf.InitTestOverrides(c)

		ctx := pipelinetest.InitSelfPushTransfer(c, ProtocolR66, cliConf, partConf, servConf)
		realAddr := ctx.Server.Address

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		ctx.Server.Address = fakeAddr
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		ctx.StartService(c)

		Convey("Given a new r66 transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, ok := pip.Client.(*transferClient)
				So(ok, ShouldBeTrue)

				So(cli.Request(), ShouldBeNil)

				defer func() {
					cli.ses.Close()
					cli.conns.Done(fakeAddr)

					cont, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					_ = ctx.ServerService.Stop(cont)
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.conns.Exists(realAddr), ShouldBeTrue)
				})
			})
		})
	})
}
