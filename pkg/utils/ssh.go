package utils

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

// ParseSSHAuthorizedKey parses the given SSH public key in the `authorized_keys`
// file format, and returns it as an ssh.PublicKey instance. This function is a
// shortcut for the standard library to avoid the dogsled issues the original
// function has.
func ParseSSHAuthorizedKey(publicKey string) (ssh.PublicKey, error) {
	//nolint:dogsled // this is caused by the design of a third party library
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the SSH public key: %w", err)
	}

	return key, nil
}
