package sftp

import (
	"bytes"
	"fmt"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

func makeServerConf(db *database.DB, protoConfig *config.SftpProtoConfig,
	agent *model.LocalAgent) *ssh.ServerConfig {
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

			err2 := bcrypt.CompareHashAndPassword(user.PasswordHash, pass)
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
	agent *model.LocalAgent) (*ssh.ServerConfig, error) {
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
			l.Warningf("An error occurred while replying to a request: %v", err)
		}
	}
}

// getRule returns the rule matching the given path & direction. It also checks
// is the given account has the rights to use said rule.
func getRule(db *database.DB, logger *log.Logger, acc *model.LocalAccount,
	rulePath string, isSend bool) (*model.Rule, error) {
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

		logger.Errorf("Failed to retrieve rule: %v", err)

		return nil, fmt.Errorf("failed to retrieve rule: %w", err)
	}

	ok, err := rule.IsAuthorized(db, acc)
	if err != nil {
		logger.Errorf("Failed to check rule permissions: %v", err)

		return nil, err
	}

	if !ok {
		return nil, errRuleForbidden
	}

	return &rule, nil
}

// getListRule returns the rule associated with the given rule path, if the given
// account is authorized to use it. If 2 rules have the same path, the sending
// rule has priority.
func getListRule(db *database.DB, logger *log.Logger, acc *model.LocalAccount,
	rulePath string) (*model.Rule, error) {
	sndRule, err := getRule(db, logger, acc, rulePath, true)
	if err != nil {
		return nil, err
	}

	rcvRule, err := getRule(db, logger, acc, rulePath, false)
	if err != nil {
		return nil, err
	}

	if sndRule == nil && rcvRule == nil {
		logger.Infof("No rule found with path '%s'", rulePath)

		return nil, sftp.ErrSSHFxNoSuchFile
	}

	return sndRule, nil
}

// getRulesPaths returns the paths of all the rules which the given account has
// access to. Used for file listing purposes.
func getRulesPaths(db *database.DB, logger *log.Logger, ag *model.LocalAgent,
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
