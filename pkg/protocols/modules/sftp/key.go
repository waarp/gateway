package sftp

import (
	"bytes"
	"errors"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
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
