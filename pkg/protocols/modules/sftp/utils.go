package sftp

import (
	"golang.org/x/crypto/ssh"
)

func ParseAuthorizedKey(pubKey []byte) (ssh.PublicKey, error) {
	//nolint:dogsled //this is caused by the design of a third party library
	key, _, _, _, err := ssh.ParseAuthorizedKey(pubKey)

	//nolint:wrapcheck //this function is just a shortcut, so we leave the error as-is
	return key, err
}
