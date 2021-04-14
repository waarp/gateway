package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

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

		hostKey := &model.Crypto{
			OwnerType:  agent.TableName(),
			OwnerID:    agent.ID,
			Name:       "local_agent_key",
			PrivateKey: rsaPK,
		}
		So(db.Insert(hostKey).Run(), ShouldBeNil)

		server := NewService(db, agent, logger)
		So(server.Start(), ShouldBeNil)
		Reset(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			c.So(server.Stop(ctx), ShouldBeNil)
		})

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
				ServerCryptos: []model.Crypto{{
					Name:         "server_key",
					SSHPublicKey: rsaPBK,
				}},
				ClientCryptos: nil,
			}
			c, err := NewClient(info, make(chan model.Signal))
			So(err, ShouldBeNil)
			client := c.(*Client)
			So(client.Connect(), ShouldBeNil)

			Convey("Given the SFTP client has the same configured algos", func() {
				Convey("Then it should authenticate without errors", func() {
					So(client.Authenticate(), ShouldBeNil)
				})
			})

			Convey("Given the SFTP client has different key exchange algos", func() {
				client.conf.KeyExchanges = []string{"diffie-hellman-group1-sha1"}

				Convey("Then the authentication should fail", func() {
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})

			Convey("Given the SFTP client has different cipher algos", func() {
				client.conf.Ciphers = []string{"aes192-ctr"}

				Convey("Then the authentication should fail", func() {
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})

			Convey("Given the SFTP client has different macs algos", func() {
				client.conf.MACs = []string{"hmac-sha1"}

				Convey("Then the authentication should fail", func() {
					So(client.Authenticate(), ShouldNotBeNil)
				})
			})
		})
	})
}
