package sftp

import (
	"crypto/subtle"
	"fmt"
	"slices"

	"code.waarp.fr/lib/log"
	"golang.org/x/crypto/ssh"

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

func (*sshPublicKey) Validate(pbkey, _, _, _ string, _ bool) error {
	if _, err := ParseAuthorizedKey(pbkey); err != nil {
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
		//nolint:err113 //dynamic error is better here
		return nil, fmt.Errorf("unknown SSH public key type '%T'", value)
	}
}

type sshPrivateKey struct{}

func (*sshPrivateKey) CanOnlyHaveOne() bool { return false }

func (*sshPrivateKey) ToDB(plain, _ string) (encrypted, _ string, err error) {
	if encrypted, err = utils.AESCrypt(database.GCM, plain); err != nil {
		return "", "", fmt.Errorf("failed to encrypt the SSH private key: %w", err)
	}

	return encrypted, "", nil
}

func (*sshPrivateKey) FromDB(encrypted, _ string) (plain, _ string, err error) {
	if plain, err = utils.AESDecrypt(database.GCM, encrypted); err != nil {
		return "", "", fmt.Errorf("failed to decrypt the SSH private key: %w", err)
	}

	return plain, "", nil
}

func (*sshPrivateKey) Validate(pkey, _, _, _ string, _ bool) error {
	if _, err := ssh.ParsePrivateKey([]byte(pkey)); err != nil {
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
			logger.Errorf("Failed to retrieve the SSH certification authorities: %v", err)

			return false
		}

		for _, aut := range auths {
			pbk, err := ParseAuthorizedKey(aut.PublicIdentity)
			if err != nil {
				logger.Warningf("Failed to parse the SSH authority's public key: %v", err)

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
			logger.Errorf("Failed to retrieve the SSH certification authorities: %v", err)

			return false
		}

		for _, aut := range auths {
			if len(aut.ValidHosts) != 0 && !slices.Contains(aut.ValidHosts, address) {
				continue
			}

			pbk, err := ParseAuthorizedKey(aut.PublicIdentity)
			if err != nil {
				logger.Warningf("Failed to parse the SSH authority's public key: %v", err)

				continue
			}

			if subtle.ConstantTimeCompare(key.Marshal(), pbk.Marshal()) == 1 {
				return true
			}
		}

		return false
	}
}
