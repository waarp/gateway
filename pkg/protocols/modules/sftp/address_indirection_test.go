package sftp

import (
	"net"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf/conftest"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a SFTP service with an indirect address", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, SFTP, nil, nil, nil)
		realAddr := ctx.Server.Address.String()
		conftest.InitTestOverrides(c, ctx.DB)

		So(ctx.DB.Config.Overrides.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		So(ctx.Server.Address.Set(fakeAddr), ShouldBeNil)
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		serverHostkey := &model.Credential{
			LocalAgentID: ctx.Server.NullableID(),
			Type:         AuthSSHPrivateKey,
			Name:         "sftp_hostkey",
			Value:        RSAPk,
		}
		partnerHostkey := &model.Credential{
			RemoteAgentID: ctx.Partner.NullableID(),
			Type:          AuthSSHPublicKey,
			Name:          "sftp_hostkey",
			Value:         SSHPbk,
		}
		ctx.AddCreds(c, serverHostkey, partnerHostkey)

		ctx.StartService(c)

		Convey("Given a new SFTP transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := controller.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				dialer := &protoutils.TraceDialer{Dialer: &net.Dialer{}}
				sftpClient := ctx.ClientService.(*client)
				conns := protoutils.NewConnPool[*ClientConn](dialer, sftpClient.newClientConn)
				cli := newTransferClient(pip.Pip, conns)

				So(cli.Request(), ShouldBeNil)

				defer func() {
					cli.conns.CloseConnFor(ctx.RemAccount)
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.sftpSession.ssh.RemoteAddr().String(), ShouldEqual, realAddr)
				})
			})
		})
	})
}
