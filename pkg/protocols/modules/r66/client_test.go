package r66

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestGetConnection(t *testing.T) {
	Convey("Given an R66 client", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
		ctx.AddCreds(c, serverPassword(ctx.Server), partnerPassword(ctx.Partner))
		ctx.StartService(c)

		r66client, isR66 := ctx.ClientService.(*Client)
		So(isR66, ShouldBeTrue)

		r66client.conns.SetGracePeriod(time.Millisecond)

		Convey(`When calling the "GetConnection" method`, func() {
			conn, connErr := r66client.GetConnection(ctx.Partner, ctx.RemAccount)
			SoMsg("Then the error should be nil", connErr, ShouldBeNil)

			defer conn.Close()

			Convey("Then the returned connection should be valid", func() {
				ses, sesErr := conn.NewSession()
				So(sesErr, ShouldBeNil)

				defer ses.Close()
			})

			Convey("Then the connection counter should have increased", func() {
				So(r66client.conns.Exists(ctx.RemAccount), ShouldBeTrue)
			})

			Convey("When returning the connection", func() {
				r66client.ReturnConnection(ctx.RemAccount)

				Convey("Then the connection should be returned to the pool", func() {
					<-time.After(100 * time.Millisecond)
					So(r66client.conns.Exists(ctx.RemAccount), ShouldBeFalse)
				})
			})
		})
	})
}
