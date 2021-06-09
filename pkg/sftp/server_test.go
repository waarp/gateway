package sftp

import (
	"context"
	"encoding/json"
	"net"
	"path"
	"path/filepath"
	"testing"
	"time"

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

		hostKey := &model.Crypto{
			OwnerType:  agent.TableName(),
			OwnerID:    agent.ID,
			Name:       "test_sftp_server_key",
			PrivateKey: rsaPK,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

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

func TestSSHServerInterruption(t *testing.T) {

	Convey("Given an SFTP server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, "sftp", servConf)
		test.AddCerts(c, makeServerKey(test.Server))

		serv := gatewayd.ServiceConstructors[test.Server.Protocol](test.DB, test.Server, test.Logger)
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy SFTP client", func() {
			cli := makeDummyClient(test.Server.Address, pipelinetest.TestLogin, pipelinetest.TestPassword)

			Convey("Given that a push transfer started", func() {
				dst, err := cli.Create(path.Join(test.Rule.Path, "test_in_shutdown.dst"))
				So(err, ShouldBeNil)

				_, err = dst.Write([]byte("abc"))
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					So(serv.Stop(ctx), ShouldBeNil)

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
							Filesize:   -1,
							RuleID:     test.Rule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      database.Owner,
							Progress:   3,
						}
						So(transfers[0], ShouldResemble, trans)
					})
				})
			})
		})
	})

	Convey("Given an SFTP server ready for pull transfers", t, func(c C) {
		test := pipelinetest.InitServerPull(c, "sftp", servConf)
		test.AddCerts(c, makeServerKey(test.Server))

		serv := gatewayd.ServiceConstructors[test.Server.Protocol](test.DB, test.Server, test.Logger)
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy SFTP client", func() {
			cli := makeDummyClient(test.Server.Address, pipelinetest.TestLogin, pipelinetest.TestPassword)

			Convey("Given that a pull transfer started", func() {
				pipelinetest.AddSourceFile(c, filepath.Join(test.Server.Root,
					test.Server.LocalOutDir), "test_out_shutdown.src")
				dst, err := cli.Open(path.Join(test.Rule.Path, "test_out_shutdown.src"))
				So(err, ShouldBeNil)

				_, err = dst.Read(make([]byte, 3))
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					So(serv.Stop(ctx), ShouldBeNil)

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
							Filesize:   int64(pipelinetest.TestFileSize),
							RuleID:     test.Rule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      database.Owner,
							Progress:   3,
						}
						So(transfers[0], ShouldResemble, trans)
					})
				})
			})
		})
	})
}
