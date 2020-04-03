package sftp

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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

func getSSHConfig(certs []model.Cert, a *model.RemoteAccount) (*ssh.ClientConfig, error) {
	pwd, err := model.DecryptPassword(a.Password)
	if err != nil {
		return nil, err
	}

	keys := []ssh.PublicKey{}
	types := map[string]struct{}{}
	for _, c := range certs {
		key, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey) //nolint:dogsled
		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
		types[key.Type()] = struct{}{}
	}
	algos := make([]string, len(types))
	i := 0
	for k := range types {
		algos[i] = k
		i++
	}

	return &ssh.ClientConfig{
		User: a.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback:   makeFixedHostKeys(keys),
		HostKeyAlgorithms: algos,
	}, nil
}
