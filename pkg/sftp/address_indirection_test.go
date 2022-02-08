package sftp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "not_a_real_address:99999"

	Convey("Given a SFTP service with an indirect address", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		defer func() { pipeline.TestPipelineEnd = nil }()

		realAddr := ctx.Server.Address
		conf.InitTestOverrides(c)

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		ctx.Server.Address = fakeAddr
		ctx.Partner.Address = fakeAddr
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)
		So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

		serverHostkey := model.Crypto{
			OwnerType:  ctx.Server.TableName(),
			OwnerID:    ctx.Server.ID,
			Name:       "sftp_hostkey",
			PrivateKey: rsaPK,
		}
		partnerHostkey := model.Crypto{
			OwnerType:    ctx.Partner.TableName(),
			OwnerID:      ctx.Partner.ID,
			Name:         "sftp_hostkey",
			SSHPublicKey: rsaPBK,
		}
		ctx.AddCryptos(c, serverHostkey, partnerHostkey)

		ctx.StartService(c)

		Convey("Given a new SFTP transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := newClient(pip.Pip)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				defer func() {
					_ = cli.remoteFile.Close()
					_ = cli.sftpSession.Close()
					_ = cli.sshSession.Close()
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.sshSession.RemoteAddr().String(), ShouldEqual, realAddr)
				})
			})
		})
	})
}
