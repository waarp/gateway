package sftp

import (
	"bytes"
	"errors"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var (
	errSSHNoKey       = errors.New("no key found")
	errSSHKeyMismatch = errors.New("the SSH key does not match known keys")
)

type fixedHostKeys []ssh.PublicKey

func (f fixedHostKeys) check(_ string, _ net.Addr, remoteKey ssh.PublicKey) error {
	if len(f) == 0 {
		return fmt.Errorf("ssh: required host key was nil: %w", errSSHNoKey)
	}

	remoteBytes := remoteKey.Marshal()
	for _, key := range f {
		if bytes.Equal(remoteBytes, key.Marshal()) {
			return nil
		}
	}

	return fmt.Errorf("ssh: host key mismatch: %w", errSSHKeyMismatch)
}

func makeFixedHostKeys(keys []ssh.PublicKey) ssh.HostKeyCallback {
	hk := fixedHostKeys(keys)

	return hk.check
}

func getSSHClientConfig(info *model.TransferContext, protoConfig *config.SftpPartnerProtoConfig,
) (*ssh.ClientConfig, error) {
	var (
		hostKeys []ssh.PublicKey
		algos    []string
	)

	for _, crypto := range info.RemoteAgentCryptos {
		key, err := utils.ParseSSHAuthorizedKey(crypto.SSHPublicKey)
		if err != nil {
			return nil, fmt.Errorf("cannot parse server key: %w", err)
		}

		hostKeys = append(hostKeys, key)

		if !utils.ContainsStrings(algos, key.Type()) {
			algos = append(algos, key.Type())
		}
	}

	conf := &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: protoConfig.KeyExchanges,
			Ciphers:      protoConfig.Ciphers,
			MACs:         protoConfig.MACs,
		},
		User:              info.RemoteAccount.Login,
		Auth:              []ssh.AuthMethod{},
		HostKeyCallback:   makeFixedHostKeys(hostKeys),
		HostKeyAlgorithms: algos,
	}

	var signers []ssh.Signer

	for _, crypto := range info.RemoteAccountCryptos {
		signer, err := ssh.ParsePrivateKey([]byte(crypto.PrivateKey))
		if err != nil {
			continue
		}

		signers = append(signers, signer)
	}

	if len(signers) > 0 {
		conf.Auth = append(conf.Auth, ssh.PublicKeys(signers...))
	}

	if len(info.RemoteAccount.Password) > 0 {
		conf.Auth = append(conf.Auth, ssh.Password(string(info.RemoteAccount.Password)))
	}

	setDefaultClientAlgos(conf)

	return conf, nil
}

func setDefaultClientAlgos(conf *ssh.ClientConfig) {
	if len(conf.KeyExchanges) == 0 {
		conf.KeyExchanges = config.SFTPValidKeyExchanges.ClientDefaults()
	}

	if len(conf.Ciphers) == 0 {
		conf.Ciphers = config.SFTPValidCiphers.ClientDefaults()
	}

	if len(conf.MACs) == 0 {
		conf.MACs = config.SFTPValidMACs.ClientDefaults()
	}
}
