package internal

import (
	"bytes"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

// GetSSHServerConfig builds and returns an ssh.ServerConfig from the given
// parameters. By default, the server accepts both public key & password
// authentication, with the former having priority over the later.
func GetSSHServerConfig(db *database.DB, certs []model.Crypto, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent) (*ssh.ServerConfig, error) {
	conf := &ssh.ServerConfig{
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
					return nil, fmt.Errorf("internal database error")
				}
				return nil, fmt.Errorf("authentication failed")
			}
			certs, err := user.GetCryptos(db)
			if err != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			for _, cert := range certs {
				publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cert.SSHPublicKey))
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
			err1 := db.Get(&user, "local_agent_id=? AND login=?", agent.ID,
				conn.User()).Run()
			if err1 != nil {
				if !database.IsNotFound(err1) {
					return nil, fmt.Errorf("internal database error")
				}
			}
			err2 := bcrypt.CompareHashAndPassword(user.PasswordHash, pass)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			return &ssh.Permissions{}, nil
		},
	}

	for _, cert := range certs {
		privateKey, err := ssh.ParsePrivateKey([]byte(cert.PrivateKey))
		if err != nil {
			return nil, err
		}
		conf.AddHostKey(privateKey)
	}

	return conf, nil
}

// AcceptRequests accepts all SFTP requests received on the given channel, and
// rejects all other types of SSH requests.
func AcceptRequests(in <-chan *ssh.Request) {
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

// GetRule returns the rule matching the given path & direction. It also checks
// is the given account has the rights to use said rule.
func GetRule(db *database.DB, logger *log.Logger, acc *model.LocalAccount,
	ag *model.LocalAgent, rulePath string, isSend bool) (*model.Rule, error) {

	var rule model.Rule
	if err := db.Get(&rule, "path=? AND send=?", rulePath, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			direction := "receive"
			if isSend {
				direction = "sending"
			}
			logger.Debugf("No %s rule found for path '%s'", direction, rulePath)
			return nil, nil
		}
		logger.Errorf("Failed to retrieve rule: %s", err)
		return nil, fmt.Errorf("failed to retrieve rule: %s", err)
	}

	var accesses model.RuleAccesses
	if err := db.Select(&accesses).Where("rule_id=?", rule.ID).Run(); err != nil {
		logger.Errorf("Failed to retrieve rule permissions: %s", err)
		return nil, fmt.Errorf("failed to retrieve rule permissions")
	}

	if len(accesses) == 0 {
		return &rule, nil
	}

	for _, access := range accesses {
		if (access.ObjectType == "local_agents" && access.ObjectID == ag.ID) ||
			(access.ObjectType == "local_accounts" && access.ObjectID == acc.ID) {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("user is not allowed to use the specified rule")
}

// GetListRule returns the rule associated with the given rule path, if the given
// account is authorized to use it. If 2 rules have the same path, the sending
// rule has priority.
func GetListRule(db *database.DB, logger *log.Logger, acc *model.LocalAccount,
	ag *model.LocalAgent, rulePath string) (*model.Rule, error) {
	sndRule, err := GetRule(db, logger, acc, ag, rulePath, true)
	if err != nil {
		return nil, err
	}

	rcvRule, err := GetRule(db, logger, acc, ag, rulePath, false)
	if err != nil {
		return nil, err
	}

	if sndRule == nil && rcvRule == nil {
		logger.Infof("No rule found with path '%s'", rulePath)
		return nil, sftp.ErrSSHFxNoSuchFile
	}
	return sndRule, nil
}

// GetRulesPaths returns the paths of all the rules which the given account has
// access to. Used for file listing purposes.
func GetRulesPaths(db *database.DB, logger *log.Logger, ag *model.LocalAgent,
	acc *model.LocalAccount) ([]string, error) {

	var rules model.Rules
	query := db.Select(&rules).Distinct("path").Where(
		`(id IN 
			(SELECT DISTINCT rule_id FROM rule_access WHERE
				(object_id=? AND object_type='local_accounts') OR
				(object_id=? AND object_type='local_agents')
			)
		)
		OR 
		( (SELECT COUNT(*) FROM rule_access WHERE rule_id = id) = 0 )`,
		acc.ID, ag.ID).OrderBy("path", true)
	if err := query.Run(); err != nil {
		logger.Errorf("Failed to retrieve rule list: %s", err)
		return nil, err
	}

	paths := make([]string, len(rules))
	for i := range rules {
		paths[i] = rules[i].Path
	}
	return paths, nil
}
