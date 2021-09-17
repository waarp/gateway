package sftp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

var (
	errSSHNoKey       = errors.New("no key found")
	errSSHKeyMismatch = errors.New("the SSH key does not match known keys")
)

func isRemoteTaskError(err error) (string, bool) {
	if !strings.Contains(err.Error(), "TransferError(TeExternalOperation)") {
		return "", false
	}

	msg := strings.TrimPrefix(err.Error(), "sftp: \"TransferError(TeExternalOperation): ")
	msg = strings.TrimSuffix(msg, "\" (SSH_FX_FAILURE)")

	return msg, true
}

type fixedHostKeys []ssh.PublicKey

func (f fixedHostKeys) check(_ string, _ net.Addr, key ssh.PublicKey) error {
	if len(f) == 0 {
		return fmt.Errorf("ssh: required host key was nil: %w", errSSHNoKey)
	}

	remoteKey := key.Marshal()
	for _, key := range f {
		if bytes.Equal(remoteKey, key.Marshal()) {
			return nil
		}
	}

	return fmt.Errorf("ssh: host key mismatch: %w", errSSHKeyMismatch)
}

func makeFixedHostKeys(keys []ssh.PublicKey) ssh.HostKeyCallback {
	hk := fixedHostKeys(keys)

	return hk.check
}

func exist(slice []string, elem string) bool {
	for _, e := range slice {
		if e == elem {
			return true
		}
	}

	return false
}

func getSSHClientConfig(info *model.OutTransferInfo, protoConfig *config.SftpProtoConfig) (*ssh.ClientConfig, error) {
	var hostKeys []ssh.PublicKey

	var algos []string

	for i := range info.ServerCryptos {
		//nolint:dogsled // the 3rd party lib is designed this way, nothing to do about it
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(info.ServerCryptos[i].SSHPublicKey))
		if err != nil {
			return nil, fmt.Errorf("cannot parse server key: %w", err)
		}

		hostKeys = append(hostKeys, key)

		if !exist(algos, key.Type()) {
			algos = append(algos, key.Type())
		}
	}

	conf := &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		User:              info.Account.Login,
		Auth:              []ssh.AuthMethod{},
		HostKeyCallback:   makeFixedHostKeys(hostKeys),
		HostKeyAlgorithms: algos,
	}

	var signers []ssh.Signer

	for i := range info.ClientCryptos {
		signer, err := ssh.ParsePrivateKey([]byte(info.ClientCryptos[i].PrivateKey))
		if err != nil {
			continue
		}

		signers = append(signers, signer)
	}

	if len(signers) > 0 {
		conf.Auth = append(conf.Auth, ssh.PublicKeys(signers...))
	}

	if len(info.Account.Password) > 0 {
		conf.Auth = append(conf.Auth, ssh.Password(string(info.Account.Password)))
	}

	return conf, nil
}
