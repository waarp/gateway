package r66

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a r66 service with an indirect address", t, func(c C) {
		conf.InitTestOverrides(c)

		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
		ctx.AddCreds(c, serverPassword(ctx.Server), partnerPassword(ctx.Partner))

		realAddr := ctx.Server.Address.String()
		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		So(ctx.Server.Address.Set(fakeAddr), ShouldBeNil)
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		So(ctx.ServerService.Start(), ShouldBeNil)

		Convey("Given a new r66 transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := controller.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)
				Reset(func() { pip.Pip.EndTransfer() })

				trans, err := ctx.ClientService.InitTransfer(pip.Pip)
				So(err, ShouldBeNil)

				cli, ok := trans.(*transferClient)
				So(ok, ShouldBeTrue)

				So(cli.Request(), ShouldBeNil)

				defer func() {
					cli.ses.Close()
					cli.conns.CloseConnFor(ctx.RemAccount)

					cont, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					_ = ctx.ServerService.Stop(cont)
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.conns.Exists(ctx.RemAccount), ShouldBeTrue)
				})
			})
		})
	})
}
