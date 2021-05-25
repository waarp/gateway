package sftp

import (
	"context"
	"encoding/json"
	"net"
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
)

func getTestPort() string {
	listener, err := net.Listen("tcp", "localhost:0")
	So(err, ShouldBeNil)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	So(err, ShouldBeNil)
	So(listener.Close(), ShouldBeNil)

	return port
}

func TestServerStop(t *testing.T) {
	Convey("Given a running SFTP server service", t, func(dbc C) {
		db := database.TestDatabase(dbc, "ERROR")
		port := getTestPort()

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

		server := NewService(db, agent, log.NewLogger("test_sftp_server"))
		So(server.Start(), ShouldBeNil)

		Convey("When stopping the service", func() {
			err := server.Stop(context.Background())

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SFTP server should no longer respond", func() {
					_, err := ssh.Dial("tcp", "localhost:"+port, &ssh.ClientConfig{})
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestServerStart(t *testing.T) {
	Convey("Given an SFTP server service", t, func(dbc C) {
		db := database.TestDatabase(dbc, "ERROR")
		port := getTestPort()
		root, err := filepath.Abs("server_start_root")
		So(err, ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

		sftpServer := NewService(db, agent, log.NewLogger("test_sftp_server"))

		Convey("When starting the server", func() {
			err := sftpServer.Start()

			Reset(func() {
				_ = sftpServer.Stop(context.Background())
			})

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSSHServer(t *testing.T) {

	Convey("Given a server transfer context", t, func(c C) {
		testCtx := testhelpers.InitServer(c, "sftp", nil)
		testhelpers.MakeChan(c)
		hostKey := &model.Cert{
			OwnerType:  testCtx.Server.TableName(),
			OwnerID:    testCtx.Server.ID,
			Name:       "sftp_hostkey",
			PrivateKey: []byte(rsaPK),
		}
		So(testCtx.DB.Insert(hostKey).Run(), ShouldBeNil)

		Convey("Given an SFTP server", func() {
			sftpServer := NewService(testCtx.DB, testCtx.Server, log.NewLogger("test_sftp_server")).(*Service)
			So(sftpServer.Start(), ShouldBeNil)
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				So(sftpServer.Stop(ctx), ShouldBeNil)
			})

			Convey("Given that the server shuts down", func() {
				Convey("Given an SSH client", func() {
					cli := makeDummyClient(testCtx.Server.Address, testhelpers.
						TestLogin, testhelpers.TestPassword)

					Convey("Given an incoming transfer", func() {
						dst, err := cli.Create(path.Join(testCtx.ServerPush.Path,
							"test_in_shutdown.dst"))
						So(err, ShouldBeNil)

						Convey("When the server shuts down", func(c C) {
							_, err := dst.Write([]byte{'a'})
							So(err, ShouldBeNil)

							pip, ok := sftpServer.listener.runningTransfers.Get(1)
							So(ok, ShouldBeTrue)
							pip.Interrupt()

							_, err = dst.Write([]byte{'b'})
							So(err, ShouldNotBeNil)

							pipeline.WaitEndServerTransfer(c, pip.(*pipeline.ServerPipeline))

							Convey("Then the transfer should appear interrupted", func() {
								var transfers model.Transfers
								So(testCtx.DB.Select(&transfers).Run(), ShouldBeNil)
								So(transfers, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:        transfers[0].ID,
									Start:     transfers[0].Start,
									IsServer:  true,
									AccountID: testCtx.LocAccount.ID,
									AgentID:   testCtx.Server.ID,
									LocalPath: filepath.Join(testCtx.Server.Root,
										testCtx.Server.LocalTmpDir, "test_in_shutdown.dst.part"),
									RemotePath: "/test_in_shutdown.dst",
									RuleID:     testCtx.ServerPush.ID,
									Status:     types.StatusInterrupted,
									Step:       types.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(transfers[0], ShouldResemble, trans)
							})
						})
					})

					Convey("Given an outgoing transfer", func() {
						testhelpers.AddSourceFile(c, filepath.Join(testCtx.Server.Root,
							testCtx.Server.LocalOutDir), "test_out_shutdown.src")
						src, err := cli.Open(path.Join(testCtx.ServerPull.Path,
							"test_out_shutdown.src"))
						So(err, ShouldBeNil)

						Convey("When the server shuts down", func() {
							_, err := src.Read(make([]byte, 1))
							So(err, ShouldBeNil)

							pip, ok := sftpServer.listener.runningTransfers.Get(1)
							So(ok, ShouldBeTrue)
							pip.Interrupt()

							_, err = src.Read(make([]byte, 1))
							So(err, ShouldNotBeNil)

							pipeline.WaitEndServerTransfer(c, pip.(*pipeline.ServerPipeline))

							Convey("Then the transfer should appear interrupted", func() {
								var transfers model.Transfers
								So(testCtx.DB.Select(&transfers).Run(), ShouldBeNil)
								So(transfers, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:        transfers[0].ID,
									Start:     transfers[0].Start,
									IsServer:  true,
									AccountID: testCtx.LocAccount.ID,
									AgentID:   testCtx.Server.ID,
									LocalPath: filepath.Join(testCtx.Server.Root,
										testCtx.Server.LocalOutDir, "test_out_shutdown.src"),
									RemotePath: "/test_out_shutdown.src",
									RuleID:     testCtx.ServerPull.ID,
									Status:     types.StatusInterrupted,
									Step:       types.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(transfers[0], ShouldResemble, trans)
							})
						})
					})
				})
			})
		})
	})
}
