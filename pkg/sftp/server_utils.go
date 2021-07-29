package sftp

import (
	"bytes"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

func getSSHServerConfig(db *database.DB, hostKeys []model.Crypto, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent) (*ssh.ServerConfig, error) {
	conf := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			user := &model.LocalAccount{LocalAgentID: agent.ID, Login: conn.User()}
			if err := db.Get(user, "local_agent_id=? AND login=?", agent.ID,
				conn.User()).Run(); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}
			userKeys, err := user.GetCryptos(db)
			if err != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			for _, userKey := range userKeys {
				publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(userKey.SSHPublicKey))
				if err != nil {
					return nil, err
				}
				if bytes.Equal(publicKey.Marshal(), key.Marshal()) {
					return &ssh.Permissions{}, nil
				}
			}
			return nil, fmt.Errorf("authentication failed")
		},
		PasswordCallback: func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			var user model.LocalAccount
			if err := db.Get(&user, "local_agent_id=? AND login=?", agent.ID,
				conn.User()).Run(); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}
			if err := bcrypt.CompareHashAndPassword(user.PasswordHash, pass); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			return &ssh.Permissions{}, nil
		},
	}

	if len(hostKeys) == 0 {
		return nil, fmt.Errorf("'%s' SFTP server is missing a hostkey", agent.Name)
	}
	for _, hostKey := range hostKeys {
		privateKey, err := ssh.ParsePrivateKey([]byte(hostKey.PrivateKey))
		if err != nil {
			return nil, err
		}
		conf.AddHostKey(privateKey)
	}

	return conf, nil
}

func getAccount(db *database.DB, agentID uint64, login string) (*model.LocalAccount, error) {
	var account model.LocalAccount
	if err := db.Get(&account, "local_agent_id=? AND login=?", agentID, login).Run(); err != nil {
		return nil, err
	}
	return &account, nil
}

func acceptRequests(in <-chan *ssh.Request) {
	for req := range in {
		ok := false
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				ok = true
			}
		}
		_ = req.Reply(ok, nil)
	}
}
