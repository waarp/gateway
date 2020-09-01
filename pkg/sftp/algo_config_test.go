package sftp

import (
	"context"
	"fmt"
	"net"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func getFreePort() uint16 {
	l, err := net.Listen("tcp", "localhost:0")
	So(err, ShouldBeNil)

	port := uint16(l.Addr().(*net.TCPAddr).Port)
	So(l.Close(), ShouldBeNil)

	return port
}

func TestSFTPAlgoConfig(t *testing.T) {
	logger := log.NewLogger("test_sftp_algo")

	login := "login"
	password := []byte("password")

	Convey("Given an SFTP server", t, func(c C) {
		db := database.GetTestDatabase()
		root := testhelpers.TempDir(c, "algo-config")
		port := getFreePort()

		agent := &model.LocalAgent{
			Name:     "sftp_server",
			Protocol: "sftp",
			Paths: &model.ServerPaths{
				Root: root,
			},
			ProtoConfig: []byte(`{
				"address": "localhost",
				"port": ` + fmt.Sprint(port) + `,
				"keyExchanges": ["ecdh-sha2-nistp256"],
				"ciphers": ["aes128-ctr"],
				"macs": ["hmac-sha2-256"]
			}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        login,
			Password:     password,
		}
		So(db.Create(account), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "local_agent_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		server := NewService(db, agent, logger)
		So(server.Start(), ShouldBeNil)
		Reset(func() { _ = server.Stop(context.Background()) })

		Convey("Given an SFTP client", func() {
			info := model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Name:        "remote_sftp",
					Protocol:    "sftp",
					ProtoConfig: agent.ProtoConfig,
				},
				Account: &model.RemoteAccount{
					Login:    login,
					Password: password,
				},
				ServerCerts: []model.Cert{{
					Name:        "remote_agent_cert",
					PublicKey:   testPBK,
					Certificate: []byte("cert"),
				}},
				ClientCerts: nil,
			}
			c, err := NewClient(info, make(chan model.Signal))
			So(err, ShouldBeNil)
			client := c.(*Client)

			Convey("Given the SFTP client has the same configured algos", func() {
				Convey("Then it should authenticate without errors", func() {
					So(client.Connect(), ShouldBeNil)
					So(client.Authenticate(), ShouldBeNil)
				})
			})

			Convey("Given the SFTP client has different key exchange algos", func() {
				client.conf.KeyExchanges = []string{"diffie-hellman-group1-sha1"}

				Convey("Then the authentication should fail", func() {
					So(client.Connect(), ShouldBeNil)
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})

			Convey("Given the SFTP client has different cipher algos", func() {
				client.conf.Ciphers = []string{"aes192-ctr"}

				Convey("Then the authentication should fail", func() {
					So(client.Connect(), ShouldBeNil)
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})

			Convey("Given the SFTP client has different macs algos", func() {
				client.conf.MACs = []string{"hmac-sha1"}

				Convey("Then the authentication should fail", func() {
					So(client.Connect(), ShouldBeNil)
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})
		})
	})
}
