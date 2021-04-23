package sftp

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRequest(t *testing.T) {
	logger := log.NewLogger("test_client_request")

	Convey("Given a client transfer context", t, func(c C) {
		ctx := testhelpers.InitClient(c, "sftp", nil)

		Convey("Given a lambda SFTP server", func() {
			makeDummyServer(ctx.Paths.GatewayHome, ctx.Partner.Address)

			transCtx := &model.TransferContext{
				RemoteAgent:   ctx.Partner,
				RemoteAccount: ctx.RemAccount,
				Rule:          ctx.ClientPush,
				RemoteAgentCerts: []model.Cert{{
					Name:      "partner_key",
					PublicKey: []byte(rsaPBK),
				}},
				Transfer: &model.Transfer{
					IsServer:   false,
					LocalPath:  "trans_local",
					RemotePath: "trans_remote",
				},
			}

			Convey("Given a valid out file transfer", func() {
				transCtx.Rule.IsSend = true

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then it should NOT return an error", func() {
					So(cli.Request(), ShouldBeNil)

					Convey("Then the file stream should be open", func() {
						So(cli.(*client).remoteFile, ShouldNotBeNil)
						_ = cli.(*client).remoteFile.Close()
					})
				})
			})

			Convey("Given a valid in file transfer", func(c C) {
				transCtx.Rule.IsSend = false
				testhelpers.AddSourceFile(c, ctx.Paths.GatewayHome, "trans_remote")

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then it should NOT return an error", func() {
					So(cli.Request(), ShouldBeNil)

					Convey("Then the file stream should be open", func() {
						So(cli.(*client).remoteFile, ShouldNotBeNil)
						_ = cli.(*client).remoteFile.Close()
					})
				})
			})

			Convey("Given an invalid out file transfer", func() {
				transCtx.Rule.IsSend = true
				transCtx.Transfer.RemotePath = "invalid/file"

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then it should return an error", func() {
					So(cli.Request(), ShouldNotBeNil)

					Convey("Then the file stream should NOT be open", func() {
						So(cli.(*client).remoteFile, ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid in file transfer", func() {
				transCtx.Rule.IsSend = false
				transCtx.Transfer.RemotePath = "invalid"

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then it should return an error", func() {
					So(cli.Request(), ShouldNotBeNil)

					Convey("Then the file stream should NOT be open", func() {
						So(cli.(*client).remoteFile, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestData(t *testing.T) {
	logger := log.NewLogger("test_client_request")

	Convey("Given a client transfer context", t, func(c C) {
		ctx := testhelpers.InitClient(c, "sftp", nil)

		Convey("Given a lambda SFTP server", func() {
			makeDummyServer(ctx.Paths.GatewayHome, ctx.Partner.Address)

			transCtx := &model.TransferContext{
				RemoteAgent:   ctx.Partner,
				RemoteAccount: ctx.RemAccount,
				Rule:          ctx.ClientPush,
				RemoteAgentCerts: []model.Cert{{
					Name:      "partner_key",
					PublicKey: []byte(rsaPBK),
				}},
				Transfer: &model.Transfer{
					IsServer:   false,
					LocalPath:  "trans_local",
					RemotePath: "trans_remote",
				},
			}

			Convey("Given a valid out file transfer", func() {
				transCtx.Rule.IsSend = true

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				Reset(func() { _ = cli.(*client).remoteFile.Close() })

				Convey("When transferring the data", func(c C) {
					src := testhelpers.NewSrcStream(c)
					So(cli.Data(src), ShouldBeNil)

					Convey("Then it should have sent the file content", func(c C) {
						dstCont, err := ioutil.ReadFile(filepath.Join(
							ctx.Paths.GatewayHome, "trans_remote"))
						So(err, ShouldBeNil)
						So(dstCont, ShouldResemble, src.Content())
					})
				})
			})

			Convey("Given a valid in file transfer", func(c C) {
				transCtx.Rule.IsSend = false
				srcCont := testhelpers.AddSourceFile(c, ctx.Paths.GatewayHome, "trans_remote")

				cli, err := NewClient(logger, transCtx)
				So(err, ShouldBeNil)

				So(cli.Request(), ShouldBeNil)
				Reset(func() { _ = cli.(*client).remoteFile.Close() })

				Convey("When transferring the data", func(c C) {
					dst := testhelpers.NewDstStream(c)
					So(cli.Data(dst), ShouldBeNil)

					Convey("Then it should have sent the file content", func(c C) {
						So(dst.Content(), ShouldResemble, srcCont)
					})
				})
			})
		})
	})
}
