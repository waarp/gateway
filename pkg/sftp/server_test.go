package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
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

		hostKey := &model.Crypto{
			OwnerType:  agent.TableName(),
			OwnerID:    agent.ID,
			Name:       "test_sftp_server_key",
			PrivateKey: rsaPK,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

		server := NewService(db, agent, log.NewLogger("test_sftp_server"))
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

		hostKey := &model.Crypto{
			OwnerType:  agent.TableName(),
			OwnerID:    agent.ID,
			Name:       "test_sftp_server_key",
			PrivateKey: rsaPK,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

		sftpServer := NewService(db, agent, log.NewLogger("test_sftp_server")).(*Service)

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
			So(conf.AddIndirection("indirect.ex:99999", "127.0.0.1:"+port), ShouldBeNil)
			agent.Address = "indirect.ex:99999"
			So(db.Update(agent).Cols("address").Run(), ShouldBeNil)

			Convey("When starting the server", func() {
				err := sftpServer.Start()

				Reset(func() {
					_ = sftpServer.Stop(context.Background())
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
					So(err, ShouldBeError, fmt.Errorf("'%s' SFTP server is "+
						"missing a hostkey", agent.Name))
				})
			})
		})
	})
}

func TestSSHServerInterruption(t *testing.T) {

	Convey("Given an SFTP server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, "sftp", nil)
		test.AddCryptos(c, makeServerKey(test.Server))

		serv := gatewayd.ServiceConstructors[test.Server.Protocol](test.DB, test.Server, test.Logger)
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy SFTP client", func() {
			cli := makeDummyClient(test.Server.Address, pipelinetest.TestLogin, pipelinetest.TestPassword)

			Convey("Given that a push transfer started", func(c C) {
				dst, err := cli.Create(path.Join(test.ServerRule.Path, "test_in_shutdown.dst"))
				So(err, ShouldBeNil)
				test.ServerShouldHavePreTasked(c)

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

						trans := model.Transfer{
							ID:        transfers[0].ID,
							Start:     transfers[0].Start,
							IsServer:  true,
							AccountID: test.LocAccount.ID,
							AgentID:   test.Server.ID,
							LocalPath: filepath.Join(test.Server.Root,
								test.Server.LocalTmpDir, "test_in_shutdown.dst.part"),
							RemotePath: "/test_in_shutdown.dst",
							Filesize:   model.UnknownSize,
							RuleID:     test.ServerRule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      conf.GlobalConfig.GatewayName,
							Progress:   3,
						}
						So(transfers[0], ShouldResemble, trans)

						ok := serv.(*Service).listener.runningTransfers.Exists(trans.ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})

	Convey("Given an SFTP server ready for pull transfers", t, func(c C) {
		test := pipelinetest.InitServerPull(c, "sftp", nil)
		test.AddCryptos(c, makeServerKey(test.Server))

		serv := gatewayd.ServiceConstructors[test.Server.Protocol](test.DB, test.Server, test.Logger)
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy SFTP client", func() {
			cli := makeDummyClient(test.Server.Address, pipelinetest.TestLogin, pipelinetest.TestPassword)

			Convey("Given that a pull transfer started", func() {
				pipelinetest.AddSourceFile(c, filepath.Join(test.Server.Root,
					test.Server.LocalOutDir), "test_out_shutdown.src")
				dst, err := cli.Open(path.Join(test.ServerRule.Path, "test_out_shutdown.src"))
				So(err, ShouldBeNil)
				test.ServerShouldHavePreTasked(c)

				_, err = dst.Read(make([]byte, 3))
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					res := make(chan error, 1)
					go func() {
						time.Sleep(100 * time.Millisecond)
						_, err = dst.Read(make([]byte, 3))
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

						trans := model.Transfer{
							ID:        transfers[0].ID,
							Start:     transfers[0].Start,
							IsServer:  true,
							AccountID: test.LocAccount.ID,
							AgentID:   test.Server.ID,
							LocalPath: filepath.Join(test.Server.Root,
								test.Server.LocalOutDir, "test_out_shutdown.src"),
							RemotePath: "/test_out_shutdown.src",
							Filesize:   pipelinetest.TestFileSize,
							RuleID:     test.ServerRule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      conf.GlobalConfig.GatewayName,
							Progress:   3,
						}
						So(transfers[0], ShouldResemble, trans)
					})
				})
			})
		})
	})
}
