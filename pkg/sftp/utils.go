package sftp

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
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

func exist(slice []string, elem string) bool {
	for _, e := range slice {
		if e == elem {
			return true
		}
	}
	return false
}

func getSSHClientConfig(info *model.OutTransferInfo, protoConfig *config.SftpProtoConfig) (*ssh.ClientConfig, error) {
	pwd, err := utils.DecryptPassword(info.Account.Password)
	if err != nil {
		return nil, err
	}

	var hostKeys []ssh.PublicKey
	var algos []string
	for _, c := range info.ServerCerts {
		key, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey) //nolint:dogsled
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
		User: info.Account.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback:   makeFixedHostKeys(hostKeys),
		HostKeyAlgorithms: algos,
	}

	signers := []ssh.Signer{}
	for _, c := range info.ClientCerts {
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
