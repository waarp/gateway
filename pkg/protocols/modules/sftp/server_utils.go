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
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/throttle"
)

func makeServerConf(db *database.DB, logger *log.Logger,
	protoConfig *ServerConfig, agent *model.LocalAgent,
) *ssh.ServerConfig {
	certChecker := ssh.CertChecker{
		IsUserAuthority: isUserAuthority(db, logger),
		UserKeyFallback: userKeyCallback(db, logger, agent),
	}

	pubKeyCallback := withIPCheck(db, logger, agent, certChecker.Authenticate)

	conf := &ssh.ServerConfig{
		KeyExchanges:      protoConfig.KeyExchanges,
		Ciphers:           protoConfig.Ciphers,
		MACs:              protoConfig.MACs,
		PublicKeyCallback: withThrottle(db, logger, agent, pubKeyCallback),
		PasswordCallback:  passwordCallback(db, logger, agent),
	}

	setServerDefaultAlgos(conf)

	return conf
}

// throttleAttempt returns the authentication attempt of the connection in the
// limiter. Its success is recorded once the SSH handshake is complete, since
// the public key callback is also called for keys offered without a signature.
func throttleAttempt(db *database.DB, agent *model.LocalAgent, conn ssh.ConnMetadata,
) throttle.Attempt {
	return throttle.Default().Attempt(&db.Config.Security, SFTP+":"+agent.Name,
		conn.User(), conn.RemoteAddr().String())
}

// withThrottle makes the public key authentication refuse a locked attempt.
// A refused key is not counted as a failure, since clients usually offer
// several keys before the right one, and a key cannot be guessed.
func withThrottle(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
	callback func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error),
) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if throttleAttempt(db, agent, conn).Locked() {
			logger.Warningf("Authentication of account %q from %q refused: too many failed attempts",
				conn.User(), conn.RemoteAddr())

			return nil, ErrAuthFailed
		}

		return callback(conn, key)
	}
}

func userKeyCallback(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		acc, accErr := agent.GetAccount(db, conn.User())
		if accErr != nil && !database.IsNotFound(accErr) {
			logger.Errorf("Failed to retrieve user credentials: %v", accErr)

			return nil, ErrDatabase
		}

		if res, err := acc.Authenticate(db, AuthSSHPublicKey, key); err != nil {
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

		if !acc.CheckIP(logger, conn.RemoteAddr().String()) {
			return nil, ErrAuthFailed
		}

		return &ssh.Permissions{}, nil
	}
}

// withIPCheck makes the public key authentication refuse a disallowed address
// like invalid credentials, including for SSH certificates, which do not go
// through userKeyCallback.
func withIPCheck(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
	callback func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error),
) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		perms, err := callback(conn, key)
		if err != nil {
			return nil, err
		}

		if _, isCert := key.(*ssh.Certificate); !isCert {
			// User key callback already checks IP, no need to do it again.
			return perms, nil
		}

		acc, accErr := agent.GetAccount(db, conn.User())
		if accErr != nil && !database.IsNotFound(accErr) {
			logger.Errorf("Failed to retrieve user credentials: %v", accErr)

			return nil, ErrDatabase
		}

		if !acc.CheckIP(logger, conn.RemoteAddr().String()) {
			return nil, ErrAuthFailed
		}

		return perms, nil
	}
}

func passwordCallback(db *database.DB, logger *log.Logger, agent *model.LocalAgent,
) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		attempt := throttleAttempt(db, agent, conn)
		if attempt.Locked() {
			logger.Warningf("Authentication of account %q from %q refused: too many failed attempts",
				conn.User(), conn.RemoteAddr())

			return nil, ErrAuthFailed
		}

		fail := func() {
			if attempt.Fail() {
				logger.Warningf("Too many failed authentications for account %q from %q, "+
					"new attempts are temporarily refused", conn.User(), conn.RemoteAddr())
			}
		}

		acc, accErr := agent.GetAccount(db, conn.User())
		if accErr != nil && !database.IsNotFound(accErr) {
			logger.Errorf("Failed to retrieve user credentials: %v", accErr)

			return nil, ErrDatabase
		}

		if res, err := acc.Authenticate(db, auth.Password, pass); err != nil {
			logger.Errorf("Failed to authenticate account %q: %v", acc.Login, err)

			return nil, ErrInternal
		} else if !res.Success {
			logger.Warningf("Authentication failed for account %q: %v",
				conn.User(), res.Reason)
			fail()

			return nil, ErrAuthFailed
		}

		// A disallowed address fails the SSH handshake like invalid
		// credentials, so that the handshake does not tell whether the
		// password was valid. The actual reason is logged by CheckIP.
		if !acc.CheckIP(logger, conn.RemoteAddr().String()) {
			fail()

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
	protoConfig *ServerConfig, agent *model.LocalAgent,
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
