package sftp

import (
	"bytes"
	"fmt"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

func getRuleFromPath(db *database.DB, r *sftp.Request, isSend bool) (*model.Rule, error) {
	filepath := path.Dir(r.Filepath)
	filepath = path.Clean("/" + filepath)

	rule := &model.Rule{Path: filepath, IsSend: isSend}

	if err := db.Get(rule); err != nil {
		dir := "receiving"
		if isSend {
			dir = "sending"
		}
		return nil, fmt.Errorf("cannot retrieve transfer rule: the directory "+
			"'%s' is not associated to any known %s rule", filepath, dir)
	}
	return rule, nil
}

func loadCert(db *database.DB, server *model.LocalAgent) (*model.Cert, error) {
	cert := &model.Cert{OwnerType: server.TableName(), OwnerID: server.ID}
	if err := db.Get(cert); err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			return nil, fmt.Errorf("no certificate found for SFTP server '%s'",
				server.Name)
		}
		return nil, err
	}

	return cert, nil
}

func getSSHServerConfig(db *database.DB, cert *model.Cert, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent) (*ssh.ServerConfig, error) {
	conf := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			user := &model.LocalAccount{LocalAgentID: agent.ID, Login: conn.User()}
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
			user := &model.LocalAccount{LocalAgentID: agent.ID, Login: conn.User()}
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

func getAccountID(db *database.DB, agentID uint64, login string) (uint64, error) {
	account := model.LocalAccount{LocalAgentID: agentID, Login: login}
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
