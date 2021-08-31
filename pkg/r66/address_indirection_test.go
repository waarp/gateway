package r66

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "not_a_real_address:99999"

	Convey("Given a r66 service with an indirect address", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		realAddr := ctx.Server.Address
		conf.InitTestOverrides(c)

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		ctx.Server.Address = fakeAddr
		ctx.Partner.Address = fakeAddr
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)
		So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

		ctx.StartService(c)

		Convey("Given a new r66 transfer", func(c C) {

			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := NewClient(pip.Pip)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				defer clientConns.Done(fakeAddr)
				defer cli.(*client).ses.Close()

				Convey("Then it should have connected to the server", func() {
					So(clientConns.Exists(fakeAddr), ShouldBeTrue)
				})
			})
		})
	})
}
