package sftp

import (
	"bytes"
	"fmt"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"golang.org/x/crypto/ssh"
)

type fixedHostKeys []ssh.PublicKey

func (f fixedHostKeys) check(_ string, _ net.Addr, key ssh.PublicKey) error {
	if len(f) == 0 {
		return fmt.Errorf("ssh: required host key was nil")
	}

	remoteKey := key.Marshal()
	for _, key := range f {
		if bytes.Equal(remoteKey, key.Marshal()) {
			return nil
		}
	}
	return fmt.Errorf("ssh: host key mismatch")
}

func makeFixedHostKeys(keys []ssh.PublicKey) ssh.HostKeyCallback {
	hk := fixedHostKeys(keys)
	return hk.check
}

func getSSHClientConfig(info *model.TransferContext, protoConfig *config.SftpProtoConfig) (*ssh.ClientConfig, error) {
	var hostKeys []ssh.PublicKey
	var algos []string
	for _, c := range info.RemoteAgentCryptos {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.SSHPublicKey)) //nolint:dogsled
		if err != nil {
			return nil, err
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
	for _, c := range info.RemoteAccountCryptos {
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
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

	return conf, nil
}
