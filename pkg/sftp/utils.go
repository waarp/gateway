package sftp

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"golang.org/x/crypto/ssh"
)

func getSSHConfig(c model.Cert, a model.RemoteAccount) (*ssh.ClientConfig, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey) //nolint:dogsled
	if err != nil {
		return nil, err
	}
	pwd, err := model.DecryptPassword(a.Password)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: a.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}, nil
}
