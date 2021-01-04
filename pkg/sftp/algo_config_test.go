package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSFTPAlgoConfig(t *testing.T) {
	logger := log.NewLogger("test_sftp_algo")

	login := "login"
	password := []byte("password")

	Convey("Given an SFTP server", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		root := testhelpers.TempDir(c, "algo-config")
		port := testhelpers.GetFreePort(c)

		agent := &model.LocalAgent{
			Name:     "sftp_server",
			Protocol: "sftp",
			Root:     root,
			ProtoConfig: json.RawMessage(`{
				"keyExchanges": ["ecdh-sha2-nistp256"],
				"ciphers": ["aes128-ctr"],
				"macs": ["hmac-sha2-256"]
			}`),
			Address: fmt.Sprintf("localhost:%d", port),
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        login,
			Password:     password,
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "local_agent_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

		server := NewService(db, agent, logger)
		So(server.Start(), ShouldBeNil)
		Reset(func() { _ = server.Stop(context.Background()) })

		Convey("Given an SFTP client", func() {
			info := model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Name:        "remote_sftp",
					Address:     agent.Address,
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
