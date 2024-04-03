package sftp

import (
	"golang.org/x/crypto/ssh"
)

func ParseAuthorizedKey(in string) (ssh.PublicKey, error) {
	//nolint:dogsled // this is caused by the design of a third party library
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(in))

	//nolint:wrapcheck //this is just a shortcut function; so, for better
	// error message readability, we return the error as-is
	return key, err
}

func ParsePrivateKey(in string) (ssh.Signer, error) {
	//nolint:wrapcheck //this is just a shortcut function
	return ssh.ParsePrivateKey([]byte(in))
}
