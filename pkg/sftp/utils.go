package sftp

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"golang.org/x/crypto/ssh"
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

func (f fixedHostKeys) check(_ string, _ net.Addr, remoteKey ssh.PublicKey) error {
	if len(f) == 0 {
		return fmt.Errorf("ssh: required host key was nil")
	}

	remoteBytes := remoteKey.Marshal()
	for _, key := range f {
		if bytes.Equal(remoteBytes, key.Marshal()) {
			return nil
		}
	}
	return fmt.Errorf("ssh: host key mismatch")
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
	for _, c := range info.ServerCryptos {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.SSHPublicKey)) //nolint:dogsled
		if err != nil {
			return nil, err
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
	for _, c := range info.ClientCryptos {
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
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
