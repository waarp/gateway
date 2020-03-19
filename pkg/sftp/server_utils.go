package sftp

import (
	"bytes"
	"encoding/json"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

func loadCert(db *database.DB, server *model.LocalAgent) (*model.Cert, error) {
	cert := &model.Cert{OwnerType: server.TableName(), OwnerID: server.ID}
	if err := db.Get(cert); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("no certificate found for SFTP server '%s'",
				server.Name)
		}
		return nil, err
	}

	return cert, nil
}

func gertSSHServerConfig(db *database.DB, cert *model.Cert, protoConfig *config.SftpProtoConfig) (*ssh.ServerConfig, error) {
	conf := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			user := &model.LocalAccount{Login: conn.User()}
			if err := db.Get(user); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}
			certs, err := user.GetCerts(db)
			if err != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			for _, c := range certs {
				publicKey, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey)
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
			user := &model.LocalAccount{Login: conn.User()}
			if err := db.Get(user); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}
			if err := bcrypt.CompareHashAndPassword(user.Password, pass); err != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			return &ssh.Permissions{}, nil
		},
	}

	privateKey, err := ssh.ParsePrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	conf.AddHostKey(privateKey)

	return conf, nil
}

func parseServerAddr(server *model.LocalAgent) (string, uint16, error) {
	conf := &config.SftpProtoConfig{}

	if err := json.Unmarshal(server.ProtoConfig, conf); err != nil {
		return "", 0, err
	}

	return conf.Address, conf.Port, nil
}

func getAccountID(db *database.DB, agentID uint64, login string) (uint64, error) {
	account := model.LocalAccount{
		LocalAgentID: agentID,
		Login:        login,
	}
	if err := db.Get(&account); err != nil {
		return 0, err
	}
	return account.ID, nil
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
