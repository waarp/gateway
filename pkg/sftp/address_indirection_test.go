package sftp

import (
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func TestAddressIndirection(t *testing.T) {
	fakeAddr := "9.9.9.9:9999"

	Convey("Given a SFTP service with an indirect address", t, func(c C) {
		conf.InitTestOverrides(c)

		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil, nil)
		realAddr := ctx.Server.Address

		So(conf.AddIndirection(fakeAddr, realAddr), ShouldBeNil)
		ctx.Server.Address = fakeAddr
		So(ctx.DB.Update(ctx.Server).Cols("address").Run(), ShouldBeNil)

		serverHostkey := model.Crypto{
			LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
			Name:         "sftp_hostkey",
			PrivateKey:   rsaPK,
		}
		partnerHostkey := model.Crypto{
			RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
			Name:          "sftp_hostkey",
			SSHPublicKey:  rsaPBK,
		}
		ctx.AddCryptos(c, serverHostkey, partnerHostkey)

		ctx.StartService(c)

		Convey("Given a new SFTP transfer", func(c C) {
			Convey("When connecting to the server", func(c C) {
				pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
				So(err, ShouldBeNil)

				cli, err := newTransferClient(pip.Pipeline(), &net.Dialer{}, &ssh.Config{})
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
