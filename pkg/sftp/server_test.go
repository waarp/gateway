package sftp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

var testLogConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

func TestServerStop(t *testing.T) {
	Convey("Given a running SFTP server service", t, func() {
		db := database.GetTestDatabase()
		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"test_sftp_root"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		pk, err := ioutil.ReadFile("test_sftp_root/id_rsa")
		So(err, ShouldBeNil)
		pbk, err := ioutil.ReadFile("test_sftp_root/id_rsa.pub")
		So(err, ShouldBeNil)
		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  pk,
			PublicKey:   pbk,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		server := NewServer(db, agent, log.NewLogger("test_sftp_server", testLogConf))
		err = server.Start()
		So(err, ShouldBeNil)

		Convey("When stopping the service", func() {
			err := server.Stop(context.Background())

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SFTP server should no longer respond", func() {
					_, err := ssh.Dial("tcp", "localhost:2023", &ssh.ClientConfig{})
					So(err, ShouldBeError)
				})
			})
		})
	})
}

func setupSFTPServer(pwd string) (*database.Db, *model.LocalAgent, *model.LocalAccount, *model.Cert) {
	db := database.GetTestDatabase()
	server := &model.LocalAgent{
		Name:        "test_sftp_server",
		Protocol:    "sftp",
		ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"test_sftp_root"}`),
	}
	So(db.Create(server), ShouldBeNil)

	user := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "user",
		Password:     []byte(pwd),
	}
	So(db.Create(user), ShouldBeNil)

	pk, err := ioutil.ReadFile("test_sftp_root/id_rsa")
	So(err, ShouldBeNil)
	pbk, err := ioutil.ReadFile("test_sftp_root/id_rsa.pub")
	So(err, ShouldBeNil)
	cert := &model.Cert{
		OwnerType:   server.TableName(),
		OwnerID:     server.ID,
		Name:        "test_sftp_server_cert",
		PrivateKey:  pk,
		PublicKey:   pbk,
		Certificate: []byte("cert"),
	}
	So(db.Create(cert), ShouldBeNil)

	return db, server, user, cert
}

func TestServerStart(t *testing.T) {
	Convey("Given an SFTP server service", t, func() {
		db, server, _, _ := setupSFTPServer("password")
		sftpServer := NewServer(db, server, log.NewLogger("test_sftp_server", testLogConf))

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

func getSFTPClient(pbk []byte, addr, login, pwd string) (*sftp.Client, *ssh.Client) {
	key, _, _, _, err := ssh.ParseAuthorizedKey(pbk) //nolint:dogsled
	So(err, ShouldBeNil)

	config := &ssh.ClientConfig{
		User: login,
		Auth: []ssh.AuthMethod{
			ssh.Password(pwd),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	conn, err := ssh.Dial("tcp", addr, config)
	So(err, ShouldBeNil)

	client, err := sftp.NewClient(conn)
	So(err, ShouldBeNil)

	return client, conn
}

func sftpListener(server *Server, cert *model.Cert) net.Addr {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%v", "localhost", 0))
	So(err, ShouldBeNil)

	sshConf, err := loadSSHConfig(server.db, cert)
	So(err, ShouldBeNil)

	server.listener = newListener(server.db, server.logger, listener, server.agent, sshConf)
	go server.listener.listen()
	return listener.Addr()
}

func TestSFTPServer(t *testing.T) {

	Convey("Given an SFTP server", t, func() {
		pwd := "password"
		db, agent, user, cert := setupSFTPServer(pwd)

		server := &Server{db: db, agent: agent, logger: log.NewLogger("test_sftp_server", testLogConf)}

		addr := sftpListener(server, cert)
		client, conn := getSFTPClient(cert.PublicKey, addr.String(), user.Login, pwd)

		Reset(func() {
			_ = conn.Close()
			_ = server.listener.close(context.Background())
		})

		Convey("When sending a file with SFTP", func() {
			src, err := os.Open("test_sftp_root/test_push.src")
			So(err, ShouldBeNil)

			dst, err := client.Create("/push/in/test_push.dst")
			So(err, ShouldBeNil)

			_, err = io.Copy(dst, src)
			So(err, ShouldBeNil)

			err = conn.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove("test_sftp_root/push/in/test_push.dst")
			})

			Convey("Then the destination file should exist", func() {
				_, err := os.Stat("test_sftp_root/push/in/test_push.dst")
				So(err, ShouldBeNil)

				Convey("Then the file's content should be identical to the original", func() {
					srcContent, err := ioutil.ReadFile("test_sftp_root/test_push.src")
					So(err, ShouldBeNil)
					dstContent, err := ioutil.ReadFile("test_sftp_root/push/in/test_push.dst")
					So(err, ShouldBeNil)

					So(string(dstContent), ShouldEqual, string(srcContent))
				})
			})
		})

		Convey("When requesting a file with SFTP", func() {
			src, err := client.Open("/pull/out/test_pull.src")
			So(err, ShouldBeNil)

			dst, err := os.Create("test_sftp_root/test_pull.dst")
			So(err, ShouldBeNil)

			_, err = src.WriteTo(dst)
			So(err, ShouldBeNil)

			err = conn.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove("test_sftp_root/test_pull.dst")
			})

			Convey("Then the destination file should exist", func() {
				_, err := os.Stat("test_sftp_root/test_pull.dst")
				So(err, ShouldBeNil)

				Convey("Then the file's content should be identical to the original", func() {
					srcContent, err := ioutil.ReadFile("test_sftp_root/pull/out/test_pull.src")
					So(err, ShouldBeNil)
					dstContent, err := ioutil.ReadFile("test_sftp_root/test_pull.dst")
					So(err, ShouldBeNil)

					So(string(dstContent), ShouldEqual, string(srcContent))
				})
			})
		})
	})
}
