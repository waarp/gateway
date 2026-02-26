package sftp

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

func makeServerConf(db *database.DB, logger *log.Logger,
	protoConfig *serverConfig, agent *model.LocalAgent,
) *ssh.ServerConfig {
	certChecker := ssh.CertChecker{
		IsUserAuthority: isUserAuthority(db, logger),
		UserKeyFallback: userKeyCallback(db, logger, agent),
	}

	conf := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		PublicKeyCallback: certChecker.Authenticate,
		PasswordCallback:  passwordCallback(db, logger, agent),
	}

	setServerDefaultAlgos(conf)

	return conf
}

func userKeyCallback(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		var acc model.LocalAccount
		if err := db.Get(&acc, "local_agent_id=? AND login=?", agent.ID,
			conn.User()).Run(); err != nil && !database.IsNotFound(err) {
			logger.Errorf("Failed to retrieve user credentials: %v", err)

			return nil, ErrDatabase
		}

		if res, err := acc.Authenticate(db, agent, AuthSSHPublicKey, key); err != nil {
			logger.Errorf("Failed to authenticate account %q: %v", acc.Login, err)

			return nil, ErrInternal
		} else if !res.Success {
			if acc.ID == 0 {
				logger.Warningf("Authentication failed for account %q: unknown user", conn.User())
			} else {
				logger.Warningf("Authentication failed for account %q: %v",
					conn.User(), res.Reason)
			}

			return nil, ErrAuthFailed
		}

		if len(acc.IPAddresses) > 0 {
			remoteIP := protoutils.GetIP(conn.RemoteAddr().String())
			if !acc.IPAddresses.Contains(remoteIP) {
				return nil, ErrUnauthorizedIP
			}
		}

		return &ssh.Permissions{}, nil
	}
}

func passwordCallback(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		var acc model.LocalAccount
		if err := db.Get(&acc, "local_agent_id=? AND login=?", agent.ID,
			conn.User()).Run(); err != nil && !database.IsNotFound(err) {
			logger.Errorf("Failed to retrieve user credentials: %v", err)

			return nil, ErrDatabase
		}

		if len(acc.IPAddresses) > 0 {
			remoteIP := protoutils.GetIP(conn.RemoteAddr().String())
			if !acc.IPAddresses.Contains(remoteIP) {
				return nil, ErrUnauthorizedIP
			}
		}

		if res, err := acc.Authenticate(db, agent, auth.Password, pass); err != nil {
			logger.Errorf("Failed to authenticate account %q: %v", acc.Login, err)

			return nil, ErrInternal
		} else if !res.Success {
			logger.Warningf("Authentication failed for account %q: %v",
				conn.User(), res.Reason)

			return nil, ErrAuthFailed
		}

		return &ssh.Permissions{}, nil
	}
}

func setServerDefaultAlgos(conf *ssh.ServerConfig) {
	if len(conf.KeyExchanges) == 0 {
		conf.KeyExchanges = ValidKeyExchanges.ServerDefaults()
	}

	if len(conf.Ciphers) == 0 {
		conf.Ciphers = ValidCiphers.ServerDefaults()
	}

	if len(conf.MACs) == 0 {
		conf.MACs = ValidMACs.ServerDefaults()
	}
}

// getSSHServerConfig builds and returns an ssh.ServerConfig from the given
// parameters. By default, the server accepts both public key & password
// authentication, with the former having priority over the latter.
func getSSHServerConfig(db *database.DB, logger *log.Logger, hostkeys []*model.Credential,
	protoConfig *serverConfig, agent *model.LocalAgent,
) (*ssh.ServerConfig, error) {
	conf := makeServerConf(db, logger, protoConfig, agent)

	if len(hostkeys) == 0 {
		return nil, fmt.Errorf("'%s' SFTP server is missing a hostkey: %w",
			agent.Name, errSSHNoKey)
	}

	for _, hostkey := range hostkeys {
		privateKey, err := ssh.ParsePrivateKey([]byte(hostkey.Value))
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
			if string(req.Payload[4:]) == SFTP {
				ok = true
			}
		}

		if err := req.Reply(ok, nil); err != nil {
			l.Warningf("An error occurred while replying to a request: %v", err)
		}
	}
}

func closeTCPConn(nConn net.Conn, logger *log.Logger) {
	if err := nConn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Warningf("An error occurred while closing the TCP connection: %v", err)
	}
}

func closeSSHConn(servConn *ssh.ServerConn, logger *log.Logger) {
	if err := servConn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Warningf("An error occurred while closing the SFTP connection: %v", err)
	}
}
