package sftp

import (
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a SFTP service with an indirect address", t, func(c C) {
		conf.InitTestOverrides(c)

		ctx := pipelinetest.InitSelfPushTransfer(c, SFTP, nil, nil, nil)
		realAddr := ctx.Server.Address.String()

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		So(ctx.Server.Address.Set(fakeAddr), ShouldBeNil)
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		serverHostkey := &model.Credential{
			LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
			Type:         AuthSSHPrivateKey,
			Name:         "sftp_hostkey",
			Value:        RSAPk,
		}
		partnerHostkey := &model.Credential{
			RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
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
				cli, err := newTransferClient(pip.Pip, dialer, &ssh.Config{})
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)

				defer func() {
					_ = cli.sftpFile.Close()
					_ = cli.sftpClient.Close()
					_ = cli.sshClient.Close()
				}()

				Convey("Then it should have connected to the server", func() {
					So(cli.sshClient.RemoteAddr().String(), ShouldEqual, realAddr)
				})
			})
		})
	})
}
