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
	fakeAddr := "not_a_real_address:99999"

	Convey("Given a r66 service with an indirect address", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", NewService, partConf, servConf)
		realAddr := ctx.Server.Address
		conf.InitTestOverrides(c)

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		ctx.Server.Address = fakeAddr
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		ctx.StartService(c)

		Convey("Given a new r66 transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := newClient(pip.Pipeline())
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				defer func() {
					cli.ses.Close()
					clientConns.Done(fakeAddr)

					cont, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					_ = ctx.Service().Stop(cont)
				}()

				Convey("Then it should have connected to the server", func() {
					So(clientConns.Exists(realAddr), ShouldBeTrue)
				})
			})
		})
	})
}
