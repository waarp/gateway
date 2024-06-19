package wgd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	parse "code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestChangeAESPassphrase(t *testing.T) {
	testDir := t.TempDir()

	const (
		r66ServPwd = "bar"

		remAccPwd    = "sesame"
		remAccSSHKey = testhelpers.RSAPk

		servTLSCert = testhelpers.LocalhostCert
		servTLSKey  = testhelpers.LocalhostKey

		cloudType   = "cld_typ"
		cloudSecret = "cld_sec"
	)

	filesystems.FileSystems.Store(cloudType, func(string, string, map[string]any) (fs.FS, error) {
		//nolint:nilnil //simpler for tests
		return nil, nil
	})
	defer filesystems.FileSystems.Delete(cloudType)

	Convey("Given a test database", t, func(c C) {
		db := database.TestDatabase(c)

		server := &model.LocalAgent{
			Name:     "server",
			Address:  types.Addr("localhost", 1),
			Protocol: r66.R66,
			Disabled: false,
			ProtoConfig: map[string]any{
				"serverLogin":    "foo",
				"serverPassword": r66ServPwd,
			},
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		r66Partner := &model.RemoteAgent{
			Name:     "fiz",
			Protocol: r66.R66,
			Address:  types.Addr("2.2.2.2", 2),
		}
		So(db.Insert(r66Partner).Run(), ShouldBeNil)

		r66RemAccount := &model.RemoteAccount{
			RemoteAgentID: r66Partner.ID,
			Login:         "titi",
		}
		So(db.Insert(r66RemAccount).Run(), ShouldBeNil)

		pswdCred := &model.Credential{
			RemoteAccountID: utils.NewNullInt64(r66RemAccount.ID),
			Type:            auth.Password,
			Value:           remAccPwd,
		}
		So(db.Insert(pswdCred).Run(), ShouldBeNil)

		sftpPartner := &model.RemoteAgent{
			Name:     "fuz",
			Protocol: sftp.SFTP,
			Address:  types.Addr("3.3.3.3", 3),
		}
		So(db.Insert(sftpPartner).Run(), ShouldBeNil)

		sftpRemAccount := &model.RemoteAccount{
			RemoteAgentID: sftpPartner.ID,
			Login:         "titi",
		}
		So(db.Insert(sftpRemAccount).Run(), ShouldBeNil)

		sshKeyCred := &model.Credential{
			RemoteAccountID: utils.NewNullInt64(sftpRemAccount.ID),
			Type:            sftp.AuthSSHPrivateKey,
			Value:           remAccSSHKey,
		}
		So(db.Insert(sshKeyCred).Run(), ShouldBeNil)

		tlsCertCred := &model.Credential{
			LocalAgentID: utils.NewNullInt64(server.ID),
			Type:         auth.TLSCertificate,
			Value:        servTLSCert,
			Value2:       servTLSKey,
		}
		So(db.Insert(tlsCertCred).Run(), ShouldBeNil)

		cloud := &model.CloudInstance{
			Name:   "cloud",
			Type:   cloudType,
			Secret: cloudSecret,
		}
		So(db.Insert(cloud).Run(), ShouldBeNil)

		Convey("Given a new AES passphrase", func() {
			newPassphrase := make([]byte, 32)

			_, randErr := rand.Read(newPassphrase)
			So(randErr, ShouldBeNil)

			aesFile := filepath.Join(testDir, "aes.key")
			So(os.WriteFile(aesFile, newPassphrase, 0o600), ShouldBeNil)

			newGCM := makeGCM(newPassphrase)

			Convey("When changing the AES passphrase to the new one", func() {
				configFile := filepath.Join(testDir, "config.ini")

				command := &ChangeAESPassphrase{
					ConfigFile: configFile,
					NewFile:    aesFile,
				}

				So(command.run(db), ShouldBeNil)

				Convey("Then the passphrase should have been changed", func() {
					Convey("Then the R66 server passwords should have been re-encrypted", func() {
						Convey("Both in the proto config", func() {
							row := db.QueryRow("SELECT proto_config FROM local_agents")

							var rawConfig string
							So(row.Scan(&rawConfig), ShouldBeNil)

							var protoConfig map[string]any
							So(json.Unmarshal([]byte(rawConfig), &protoConfig), ShouldBeNil)
							So(protoConfig, ShouldContainKey, "serverPassword")

							switch servPwd := protoConfig["serverPassword"].(type) {
							case string:
								pswd, newErr := utils.AESDecrypt(newGCM, servPwd)
								So(newErr, ShouldBeNil)
								So(pswd, ShouldEqual, r66ServPwd)
							default:
								So(protoConfig["serverPassword"], ShouldHaveSameTypeAs, "")
							}
						})

						Convey("And in the credentials table", func() {
							row := db.QueryRow(`SELECT value FROM credentials WHERE
                            	type=? AND local_agent_id=?`, auth.Password, server.ID)

							var cipherText string
							So(row.Scan(&cipherText), ShouldBeNil)

							pswd, aesErr := utils.AESDecrypt(newGCM, cipherText)
							So(aesErr, ShouldBeNil)
							So(pswd, ShouldEqual, r66ServPwd)
						})
					})

					Convey("Then the remote passwords should have been re-encrypted", func() {
						row := db.QueryRow(`SELECT value FROM credentials WHERE
							type=? AND remote_account_id=?`, auth.Password, r66RemAccount.ID)

						var cipherText string
						So(row.Scan(&cipherText), ShouldBeNil)

						pswd, aesErr := utils.AESDecrypt(newGCM, cipherText)
						So(aesErr, ShouldBeNil)
						So(pswd, ShouldEqual, remAccPwd)
					})

					Convey("Then the SSH private keys should have been re-encrypted", func() {
						row := db.QueryRow("SELECT value FROM credentials WHERE type=?", sftp.AuthSSHPrivateKey)

						var cipherText string
						So(row.Scan(&cipherText), ShouldBeNil)

						pswd, aesErr := utils.AESDecrypt(newGCM, cipherText)
						So(aesErr, ShouldBeNil)
						So(pswd, ShouldEqual, remAccSSHKey)
					})

					Convey("Then the TLS private keys should have been re-encrypted", func() {
						row := db.QueryRow("SELECT value2 FROM credentials WHERE type=?", auth.TLSCertificate)

						var cipherText string
						So(row.Scan(&cipherText), ShouldBeNil)

						pswd, aesErr := utils.AESDecrypt(newGCM, cipherText)
						So(aesErr, ShouldBeNil)
						So(pswd, ShouldEqual, servTLSKey)
					})

					Convey("Then the cloud secrets should have been re-encrypted", func() {
						row := db.QueryRow("SELECT secret FROM cloud_instances")

						var cipherText string
						So(row.Scan(&cipherText), ShouldBeNil)

						pswd, aesErr := utils.AESDecrypt(newGCM, cipherText)
						So(aesErr, ShouldBeNil)
						So(pswd, ShouldEqual, cloudSecret)
					})
				})

				Convey("Then the AES passphrase file should have been changed", func() {
					// both in memory
					So(conf.GlobalConfig.Database.AESPassphrase, ShouldEqual, aesFile)

					// and on disk
					serverConfig := &conf.ServerConfig{}

					parser, parsErr := parse.NewParser(serverConfig)
					So(parsErr, ShouldBeNil)

					So(parser.ParseFile(configFile), ShouldBeNil)
					So(serverConfig.Database.AESPassphrase, ShouldEqual, aesFile)
				})
			})
		})
	})
}

func makeGCM(passphrase []byte) cipher.AEAD {
	block, err := aes.NewCipher(passphrase)
	So(err, ShouldBeNil)

	gcm, err := cipher.NewGCM(block)
	So(err, ShouldBeNil)

	return gcm
}
