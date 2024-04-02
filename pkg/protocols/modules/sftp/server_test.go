package sftp

import (
	"context"
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
	Convey("Given a running SFTP server service", t, func(c C) {
		db := database.TestDatabase(c)
		port := getTestPort()

		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: SFTP,
			Address:  "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		hostKey := &model.Crypto{
			LocalAgentID: utils.NewNullInt64(agent.ID),
			Name:         "test_sftp_server_key",
			PrivateKey:   testhelpers.RSAPk,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

		server := &service{db: db, server: agent}
		So(server.Start(), ShouldBeNil)

		Convey("When stopping the service", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := server.Stop(ctx)

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
	Convey("Given an SFTP server service", t, func(c C) {
		fstest.InitMemFS(c)
		db := database.TestDatabase(c)
		port := getTestPort()
		root := "memory:/server_start_root"

		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: SFTP,
			RootDir:  root,
			Address:  "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		hostKey := &model.Crypto{
			LocalAgentID: utils.NewNullInt64(agent.ID),
			Name:         "test_sftp_server_key",
			PrivateKey:   testhelpers.RSAPk,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

		sftpServer := &service{db: db, server: agent}

		Convey("Given that the configuration is valid", func() {
			Convey("When starting the server", func() {
				err := sftpServer.Start()

				Reset(func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					So(sftpServer.Stop(ctx), ShouldBeNil)
				})

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the server address is indirect", func(c C) {
			conf.InitTestOverrides(c)
			So(conf.AddIndirection("9.9.9.9:9999", "127.0.0.1:"+port), ShouldBeNil)

			agent.Address = "9.9.9.9:9999"
			So(db.Update(agent).Cols("address").Run(), ShouldBeNil)

			Convey("When starting the server", func() {
				err := sftpServer.Start()

				Reset(func() {
					_ = sftpServer.Stop(utils.CanceledContext())
				})

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
					So(sftpServer.listener.Listener.Addr().String(), ShouldEqual,
						"127.0.0.1:"+port)
				})
			})
		})

		Convey("Given that the server is missing a hostkey", func() {
			So(db.Delete(hostKey).Run(), ShouldBeNil)

			Convey("When starting the server", func() {
				err := sftpServer.Start()

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
					So(err.Error(), ShouldContainSubstring, fmt.Sprintf("'%s' SFTP server is "+
						"missing a hostkey", agent.Name))
				})
			})
		})
	})
}

func TestSSHServerInterruption(t *testing.T) {
	Convey("Given an SFTP server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, SFTP, nil)
		test.AddCryptos(c, makeServerKey(test.Server))

		serv := &service{db: test.DB, server: test.Server}
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy SFTP client", func() {
			cli := makeDummyClient(test.Server.Address, pipelinetest.TestLogin, pipelinetest.TestPassword)

			Convey("Given that a push transfer started", func(c C) {
				dst, err := cli.Create(path.Join(test.ServerRule.Path, "test_in_shutdown.dst"))
				So(err, ShouldBeNil)

				_, err = dst.Write([]byte("123"))
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					res := make(chan error, 1)
					go func() {
						time.Sleep(100 * time.Millisecond)

						_, err = dst.Write([]byte("456"))
						res <- err
					}()

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

					defer cancel()

					So(serv.Stop(ctx), ShouldBeNil)
					So(<-res, ShouldBeError, `sftp: "TransferError(TeShuttingDown):`+
						` service is shutting down" (SSH_FX_FAILURE)`)

					Convey("Then the transfer should have been interrupted", func() {
						var transfers model.Transfers

						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)

						expected := &model.Transfer{
							ID:               transfers[0].ID,
							RemoteTransferID: transfers[0].RemoteTransferID,
							Start:            transfers[0].Start,
							LocalAccountID:   utils.NewNullInt64(test.LocAccount.ID),
							LocalPath: *mkURL(test.Paths.GatewayHome, test.Server.RootDir,
								test.ServerRule.TmpLocalRcvDir, "test_in_shutdown.dst.part"),
							DestFilename: "test_in_shutdown.dst",
							Filesize:     model.UnknownSize,
							RuleID:       test.ServerRule.ID,
							Status:       types.StatusInterrupted,
							Step:         types.StepData,
							Owner:        conf.GlobalConfig.GatewayName,
							Progress:     3,
						}
						So(transfers[0], ShouldResemble, expected)

						ok := pipeline.List.Exists(expected.ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}
