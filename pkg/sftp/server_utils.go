package sftp

import (
	"bytes"
	"fmt"

	"code.waarp.fr/lib/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

func makeServerConf(db *database.DB, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent,
) *ssh.ServerConfig {
	return &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			var user model.LocalAccount
			if err := db.Get(&user, "local_agent_id=? AND login=?", agent.ID,
				conn.User()).Run(); err != nil {
				if !database.IsNotFound(err) {
					return nil, errDatabase
				}

				return nil, errAuthFailed
			}

			certs, err := user.GetCryptos(db)
			if err != nil {
				return nil, errAuthFailed
			}

			for _, cert := range certs {
				publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cert.SSHPublicKey))
				if err != nil {
					return nil, fmt.Errorf("failed to parse public key: %w", err)
				}
				if bytes.Equal(publicKey.Marshal(), key.Marshal()) {
					return &ssh.Permissions{}, nil
				}
			}

			return nil, errAuthFailed
		},
		PasswordCallback: func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			var user model.LocalAccount

			err1 := db.Get(&user, "local_agent_id=? AND login=?", agent.ID,
				conn.User()).Run()
			if err1 != nil {
				if !database.IsNotFound(err1) {
					return nil, errDatabase
				}
			}

			err2 := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), pass)
			if err1 != nil || err2 != nil {
				return nil, errAuthFailed
			}

			return &ssh.Permissions{}, nil
		},
	}
}

// getSSHServerConfig builds and returns an ssh.ServerConfig from the given
// parameters. By default, the server accepts both public key & password
// authentication, with the former having priority over the later.
func getSSHServerConfig(db *database.DB, hostkeys []model.Crypto, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent,
) (*ssh.ServerConfig, error) {
	conf := makeServerConf(db, protoConfig, agent)

	if len(hostkeys) == 0 {
		return nil, fmt.Errorf("'%s' SFTP server is missing a hostkey: %w",
			agent.Name, errSSHNoKey)
	}

	for _, cert := range hostkeys {
		privateKey, err := ssh.ParsePrivateKey([]byte(cert.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH hostkey: %w", err)
		}

		conf.AddHostKey(privateKey)
	}

	return conf, nil
}

// acceptRequests accepts all SFTP requests received on the given channel, and
// rejects all other types of SSH requests.
func acceptRequests(in <-chan *ssh.Request, l *log.Logger) {
	for req := range in {
		ok := false

		if req.Type == "subsystem" {
			if string(req.Payload[4:]) == "sftp" {
				ok = true
			}
		}

		if err := req.Reply(ok, nil); err != nil {
			l.Warning("An error occurred while replying to a request: %v", err)
		}
	}
}
