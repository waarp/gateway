package r66

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressIndirection(t *testing.T) {
	Convey("Given a r66 service with an indirect address", t, func(c C) {
		ctx := initForSelfTransfer(c)
		addr := ctx.server.Address
		conf.InitTestOverrides(c)
		So(conf.AddIndirection("not_a_real_address:99999", addr), ShouldBeNil)
		ctx.server.Address = "not_a_real_address:99999"
		ctx.partner.Address = "not_a_real_address:99999"
		So(ctx.db.Update(ctx.server).Cols("address").Run(), ShouldBeNil)
		So(ctx.db.Update(ctx.partner).Cols("address").Run(), ShouldBeNil)

		Convey("Given a new r66 transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("When connecting to the server", func(c C) {
				info, err := model.NewOutTransferInfo(ctx.db, ctx.trans)
				So(err, ShouldBeNil)

				cli, err := NewClient(*info, nil)
				So(err, ShouldBeNil)

				So(cli.Connect(), ShouldBeNil)
				defer cli.(*client).remote.Close()

				Convey("Then it should have connected to the server", func() {
					So(cli.(*client).remote.Address, ShouldEqual, addr)
				})
			})
		})
	})
}
