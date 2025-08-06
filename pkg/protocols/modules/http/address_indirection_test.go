package http

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a HTTP service with an indirect address", t, func(c C) {
		SkipConvey("Given a new POST HTTP transfer", func(c C) {
			ctx := pipelinetest.InitSelfPushTransfer(c, HTTP, nil, nil, nil)
			realAddr := ctx.Server.Address.String()

			conf.InitTestOverrides(c)
			So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
			So(ctx.Server.Address.Set(fakeAddr), ShouldBeNil)
			So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

			ctx.StartService(c)

			Convey("When connecting to the server", func(c C) {
				pip, err := controller.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				transClient, err := ctx.ProtoClient.InitTransfer(pip.Pip)
				So(err, ShouldBeNil)

				//nolint:forcetypeassert //no need, the type assertion will always succeed
				transferClient := transClient.(*postClient)
				So(transferClient.Request(), ShouldBeNil)

				defer func() { So(transferClient.Cancel(), ShouldBeNil) }()

				Convey("Then it should have connected to the server", func() {
					So(transferClient.req.URL.Host, ShouldEqual, realAddr)
				})
			})
		})

		Convey("Given a new GET HTTP transfer", func(c C) {
			ctx := pipelinetest.InitSelfPullTransfer(c, HTTP, nil, nil, nil)
			realAddr := ctx.Server.Address.String()

			conf.InitTestOverrides(c)
			So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
			So(ctx.Server.Address.Set(fakeAddr), ShouldBeNil)
			So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

			ctx.StartService(c)

			Convey("When connecting to the server", func(c C) {
				pip, err := controller.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				transClient, err := ctx.ProtoClient.InitTransfer(pip.Pip)
				So(err, ShouldBeNil)

				//nolint:forcetypeassert //no need, the type assertion will always succeed
				transferClient := transClient.(*getClient)
				So(transferClient.Request(), ShouldBeNil)

				defer func() { transferClient.SendError(0, "") }()

				Convey("Then it should have connected to the server", func() {
					So(transferClient.resp.Request.URL.Host, ShouldEqual, realAddr)
				})
			})
		})
	})
}
