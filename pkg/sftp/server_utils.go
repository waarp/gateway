package sftp

import (
	"bytes"
	"errors"
	"fmt"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

var (
	errRuleNotFound = errors.New("transfer rule not found")
	errAuthFailed   = errors.New("authentication failed")
)

func getRuleFromPath(db *database.DB, r *sftp.Request, isSend bool) (*model.Rule, error) {
	filepath := path.Dir(r.Filepath)
	filepath = path.Clean("/" + filepath)

	rule := &model.Rule{}
	if err := db.Get(rule, "path=? AND send=?", filepath, isSend).Run(); err != nil {
		dir := "receiving"
		if isSend {
			dir = "sending"
		}

		return nil, fmt.Errorf("cannot retrieve transfer rule: the directory "+
			"'%s' is not associated to any known %s rule: %w", filepath, dir, errRuleNotFound)
	}

	return rule, nil
}

func getSSHServerConfig(db *database.DB, hostKeys []model.Crypto, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent) (*ssh.ServerConfig, error) {
	conf := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: makeVerifyPublicKey(db, agent),
		PasswordCallback:  makeVerifyPassword(db, agent),
	}

	if len(hostKeys) == 0 {
		return nil, fmt.Errorf("'%s' SFTP server is missing a hostkey: %w",
			agent.Name, errSSHNoKey)
	}

	for i := range hostKeys {
		privateKey, err := ssh.ParsePrivateKey([]byte(hostKeys[i].PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("cannot parse private key: %w", err)
		}

		conf.AddHostKey(privateKey)
	}

	return conf, nil
}

type passworkCallback func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error)

func makeVerifyPassword(db *database.DB, agent *model.LocalAgent) passworkCallback {
	return func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		var user model.LocalAccount
		if err := db.Get(&user, "local_agent_id=? AND login=?", agent.ID,
			conn.User()).Run(); err != nil {
			return nil, errAuthFailed
		}

		if err := bcrypt.CompareHashAndPassword(user.PasswordHash, pass); err != nil {
			return nil, errAuthFailed
		}

		return &ssh.Permissions{}, nil
	}
}

type publicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

func makeVerifyPublicKey(db *database.DB, agent *model.LocalAgent) publicKeyCallback {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		user := &model.LocalAccount{LocalAgentID: agent.ID, Login: conn.User()}
		if err := db.Get(user, "local_agent_id=? AND login=?", agent.ID,
			conn.User()).Run(); err != nil {
			return nil, errAuthFailed
		}

		userKeys, err := user.GetCryptos(db)
		if err != nil {
			return nil, errAuthFailed
		}

		for i := range userKeys {
			publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(userKeys[i].SSHPublicKey))
			if err != nil {
				return nil, fmt.Errorf("cannot parse partner key: %w", err)
			}

			if bytes.Equal(publicKey.Marshal(), key.Marshal()) {
				return &ssh.Permissions{}, nil
			}
		}

		return nil, errAuthFailed
	}
}

func getAccountID(db *database.DB, agentID uint64, login string) (uint64, error) {
	account := model.LocalAccount{LocalAgentID: agentID, Login: login}
	if err := db.Get(&account, "local_agent_id=? AND login=?", agentID, login).Run(); err != nil {
		return 0, err
	}

	return account.ID, nil
}

func acceptRequests(in <-chan *ssh.Request, l *log.Logger) {
	for req := range in {
		ok := false

		if req.Type == "subsystem" {
			if string(req.Payload[4:]) == "sftp" {
				ok = true
			}
		}

		if err := req.Reply(ok, nil); err != nil {
			l.Warningf("a reply operation returned an error: %v", err)
		}
	}
}
