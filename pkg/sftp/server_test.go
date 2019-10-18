package sftp

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

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

		server := &Server{Db: db, Config: agent}
		err = server.Start()
		So(err, ShouldBeNil)

		Convey("When stopping the service", func() {
			err := server.Stop(nil)

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

func TestServerStart(t *testing.T) {
	getSFTPClient := func(pbk []byte, addr, login, pwd string) *sftp.Client {
		key, _, _, _, err := ssh.ParseAuthorizedKey(pbk) //nolint:dogsled
		So(err, ShouldBeNil)

		conf := &ssh.ClientConfig{
			User: login,
			Auth: []ssh.AuthMethod{
				ssh.Password(pwd),
			},
			HostKeyCallback: ssh.FixedHostKey(key),
		}

		conn, err := ssh.Dial("tcp", addr, conf)
		So(err, ShouldBeNil)

		client, err := sftp.NewClient(conn)
		So(err, ShouldBeNil)

		return client
	}

	Convey("Given an SFTP server service", t, func() {
		db := database.GetTestDatabase()
		server := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2023, "root":"test_sftp_root"}`),
		}
		So(db.Create(server), ShouldBeNil)

		pwd := "password"
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

		sftpServer := &Server{Db: db, Config: server}

		Convey("When starting the server", func() {
			err := sftpServer.Start()

			Reset(func() {
				_ = sftpServer.Stop(nil)
			})

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				client := getSFTPClient(cert.PublicKey, "localhost:2022", user.Login, pwd)

				Convey("When sending a file with SFTP", func() {
					src, err := os.Open("test_sftp_root/test.src")
					So(err, ShouldBeNil)

					dst, err := client.Create("test_sftp_root/test.dst")
					So(err, ShouldBeNil)

					_, err = io.Copy(dst, src)
					So(err, ShouldBeNil)

					Reset(func() {
						_ = os.Remove("test_sftp_root/test.dst")
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test_sftp_root/test.dst")
						So(err, ShouldBeNil)

						Convey("Then the file's content should be identical to the original", func() {
							srcContent, err := ioutil.ReadFile("test_sftp_root/test.src")
							So(err, ShouldBeNil)
							dstContent, err := ioutil.ReadFile("test_sftp_root/test.dst")
							So(err, ShouldBeNil)

							So(string(dstContent), ShouldEqual, string(srcContent))
						})
					})
				})

				Convey("When requesting a file with SFTP", func() {
					src, err := client.Open("test_sftp_root/test.src")
					So(err, ShouldBeNil)

					dst, err := os.Create("test_sftp_root/test.dst")
					So(err, ShouldBeNil)

					_, err = io.Copy(dst, src)
					So(err, ShouldBeNil)

					Reset(func() {
						_ = os.Remove("test_sftp_root/test.dst")
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test_sftp_root/test.dst")
						So(err, ShouldBeNil)

						Convey("Then the file's content should be identical to the original", func() {
							srcContent, err := ioutil.ReadFile("test_sftp_root/test.src")
							So(err, ShouldBeNil)
							dstContent, err := ioutil.ReadFile("test_sftp_root/test.dst")
							So(err, ShouldBeNil)

							So(string(dstContent), ShouldEqual, string(srcContent))
						})
					})
				})
			})
		})
	})
}
