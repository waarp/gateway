package sftp

import (
	"context"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func addCerts(c C, ctx *testhelpers.Context) {
	serverKey := model.Cert{
		OwnerType:  ctx.Server.TableName(),
		OwnerID:    ctx.Server.ID,
		Name:       "sftp_server_cert",
		PrivateKey: []byte(rsaPK),
		PublicKey:  []byte(rsaPBK),
	}
	c.So(ctx.DB.Insert(&serverKey).Run(), ShouldBeNil)
	ctx.ServerCerts = []model.Cert{serverKey}

	partnerKey := model.Cert{
		OwnerType:  ctx.Partner.TableName(),
		OwnerID:    ctx.Partner.ID,
		Name:       "sftp_partner_cert",
		PrivateKey: []byte(rsaPK),
		PublicKey:  []byte(rsaPBK),
	}
	c.So(ctx.DB.Insert(&partnerKey).Run(), ShouldBeNil)
	ctx.ServerCerts = []model.Cert{partnerKey}
}

func startService(c C, ctx *testhelpers.Context) {
	serv := NewService(ctx.DB, ctx.Server, ctx.Logger)
	c.So(serv.Start(), ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(serv.Stop(ctx), ShouldBeNil)
	})
	serv.(*Service).listener.handlerMaker = serv.(*Service).listener.makeTestHandlers(c)
}
