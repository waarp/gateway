package sftp

import (
	"bytes"
	"fmt"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
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
	pwd, err := utils.DecryptPassword(database.GCM, info.RemoteAccount.Password)
	if err != nil {
		return nil, err
	}

	var hostKeys []ssh.PublicKey
	var algos []string
	for _, c := range info.RemoteAgentCerts {
		key, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey) //nolint:dogsled
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
		User: info.RemoteAccount.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback:   makeFixedHostKeys(hostKeys),
		HostKeyAlgorithms: algos,
	}

	var signers []ssh.Signer
	for _, c := range info.RemoteAccountCerts {
		signer, err := ssh.ParsePrivateKey(c.PrivateKey)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}
	if len(signers) > 0 {
		conf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
			ssh.Password(string(pwd)),
		}
	}

	return conf, nil
}
