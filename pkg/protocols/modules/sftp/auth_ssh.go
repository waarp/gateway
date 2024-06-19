package sftp

import (
	"crypto/subtle"
	"fmt"

	"code.waarp.fr/lib/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	AuthSSHPublicKey  = "ssh_public_key"
	AuthSSHPrivateKey = "ssh_private_key"
	AuthoritySSHCert  = "ssh_cert_authority"
)

//nolint:gochecknoinits //needed to add credential types
func init() {
	authentication.AddInternalCredentialTypeForProtocol(AuthSSHPublicKey, SFTP, &sshPublicKey{})
	authentication.AddExternalCredentialTypeForProtocol(AuthSSHPrivateKey, SFTP, &sshPrivateKey{})

	authentication.AddAuthorityType(AuthoritySSHCert, &sshCertAuthority{})
}

type sshPublicKey struct{}

func (*sshPublicKey) CanOnlyHaveOne() bool { return false }

func (*sshPublicKey) Validate(value, value2, protocol, host string, isServer bool) error {
	if _, err := ParseAuthorizedKey(value); err != nil {
		return fmt.Errorf("failed to parse SSH public key: %w", err)
	}

	return nil
}

func (*sshPublicKey) Authenticate(db database.ReadAccess, owner authentication.Owner,
	val any,
) (*authentication.Result, error) {
	switch value := val.(type) {
	case ssh.PublicKey:
		var keys model.Credentials
		if err := db.Select(&keys).Where("type=?", AuthSSHPublicKey).Where(
			owner.GetCredCond()).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve the SSH public keys: %w", err)
		}

		for _, key := range keys {
			ref, err := ParseAuthorizedKey(key.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SSH public key: %w", err)
			}

			if subtle.ConstantTimeCompare(ref.Marshal(), value.Marshal()) == 1 {
				return authentication.Success(), nil
			}
		}

		return authentication.Failure("unknown SSH public key"), nil
	default:
		//nolint:goerr113 //dynamic error is better here
		return nil, fmt.Errorf("unknown SSH public key type '%T'", value)
	}
}

type sshPrivateKey struct{}

func (*sshPrivateKey) CanOnlyHaveOne() bool { return false }

func (*sshPrivateKey) ToDB(val, _ string) (string, string, error) {
	encrypted, err := utils.AESCrypt(database.GCM, val)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt the SSH private key: %w", err)
	}

	return encrypted, "", nil
}

func (*sshPrivateKey) FromDB(val, _ string) (string, string, error) {
	clear, err := utils.AESDecrypt(database.GCM, val)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt the SSH private key: %w", err)
	}

	return clear, "", nil
}

func (*sshPrivateKey) Validate(value, value2, protocol, host string, isServer bool) error {
	if _, err := ssh.ParsePrivateKey([]byte(value)); err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	return nil
}

type sshCertAuthority struct{}

func (*sshCertAuthority) Validate(identity string) error {
	if _, err := ParseAuthorizedKey(identity); err != nil {
		return fmt.Errorf("failed to parse SSH authority public key: %w", err)
	}

	return nil
}

func isUserAuthority(db database.ReadAccess, logger *log.Logger) func(ssh.PublicKey) bool {
	return func(key ssh.PublicKey) bool {
		var auths model.Authorities
		if err := db.Select(&auths).Where("type=?", AuthoritySSHCert).Run(); err != nil {
			logger.Error("Failed to retrieve the SSH certification authorities: %s", err)

			return false
		}

		for _, aut := range auths {
			pbk, err := ParseAuthorizedKey(aut.PublicIdentity)
			if err != nil {
				logger.Warning("Failed to parse the SSH authority's public key: %s", err)

				continue
			}

			if subtle.ConstantTimeCompare(key.Marshal(), pbk.Marshal()) == 1 {
				return true
			}
		}

		return false
	}
}

func isHostAuthority(db database.ReadAccess, logger *log.Logger,
) func(key ssh.PublicKey, address string) bool {
	return func(key ssh.PublicKey, address string) bool {
		var auths model.Authorities
		if err := db.Select(&auths).Where("type=?", AuthoritySSHCert).Run(); err != nil {
			logger.Error("Failed to retrieve the SSH certification authorities: %s", err)

			return false
		}

		for _, aut := range auths {
			if len(aut.ValidHosts) != 0 && !slices.Contains(aut.ValidHosts, address) {
				continue
			}

			pbk, err := ParseAuthorizedKey(aut.PublicIdentity)
			if err != nil {
				logger.Warning("Failed to parse the SSH authority's public key: %s", err)

				continue
			}

			if subtle.ConstantTimeCompare(key.Marshal(), pbk.Marshal()) == 1 {
				return true
			}
		}

		return false
	}
}
