package http

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "not_a_real_address:99999"

	Convey("Given a HTTP service with an indirect address", t, func(c C) {
		Convey("Given a new POST HTTP transfer", func(c C) {
			ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
			defer func() { pipeline.TestPipelineEnd = nil }()

			realAddr := ctx.Server.Address
			conf.InitTestOverrides(c)

			So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
			ctx.Server.Address = fakeAddr
			ctx.Partner.Address = fakeAddr
			So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

			ctx.StartService(c)

			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := newClient(pip.Pip, nil)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				defer func() {
					_ = cli.(*postClient).Cancel()
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.(*postClient).req.URL.Host, ShouldEqual, realAddr)
				})
			})
		})

		Convey("Given a new GET HTTP transfer", func(c C) {
			ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
			defer func() { pipeline.TestPipelineEnd = nil }()

			realAddr := ctx.Server.Address
			conf.InitTestOverrides(c)

			So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
			ctx.Server.Address = fakeAddr
			ctx.Partner.Address = fakeAddr
			So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

			ctx.StartService(c)

			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := newClient(pip.Pip, nil)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				defer cli.(*getClient).SendError(nil)

				Convey("Then it should have connected to the server", func() {
					So(cli.(*getClient).resp.Request.URL.Host, ShouldEqual, realAddr)
				})
			})
		})
	})
}
