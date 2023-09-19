package sftp

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp/internal"
)

//nolint:gochecknoglobals // global var is used by design
var (
	validKeyExchanges = internal.Algos{
		{Name: "curve25519-sha256@libssh.org", ValidFor: internal.Both},
		{Name: "ecdh-sha2-nistp256", ValidFor: internal.Both},
		{Name: "ecdh-sha2-nistp384", ValidFor: internal.Both},
		{Name: "ecdh-sha2-nistp521", ValidFor: internal.Both},
		{Name: "diffie-hellman-group-exchange-sha256", ValidFor: internal.OnlyClient},
		{Name: "diffie-hellman-group1-sha1", ValidFor: internal.Both},  // Deprecated: uses SHA-1.
		{Name: "diffie-hellman-group14-sha1", ValidFor: internal.Both}, // Deprecated: uses SHA-1.
	}
	validCiphers = internal.Algos{
		{Name: "aes128-gcm@openssh.com", ValidFor: internal.Both},
		{Name: "chacha20-poly1305@openssh.com", ValidFor: internal.Both},
		{Name: "aes128-ctr", ValidFor: internal.Both},
		{Name: "aes192-ctr", ValidFor: internal.Both},
		{Name: "aes256-ctr", ValidFor: internal.Both},
		{Name: "arcfour256", ValidFor: internal.Both, DisabledByDefault: true},
		{Name: "arcfour128", ValidFor: internal.Both, DisabledByDefault: true},
		{Name: "arcfour", ValidFor: internal.Both, DisabledByDefault: true},
		{Name: "aes128-cbc", ValidFor: internal.Both, DisabledByDefault: true},
		{Name: "3des-cbc", ValidFor: internal.Both, DisabledByDefault: true},
	}
	validMACs = internal.Algos{
		{Name: "hmac-sha2-256-etm@openssh.com", ValidFor: internal.Both},
		{Name: "hmac-sha2-256", ValidFor: internal.Both},
		{Name: "hmac-sha1", ValidFor: internal.Both},    // Deprecated: uses SHA-1.
		{Name: "hmac-sha1-96", ValidFor: internal.Both}, // Deprecated: uses SHA-1.
	}
)

var (
	ErrUnknownKeyExchange = errors.New("unknown key exchange algorithm")
	ErrUnknownCipher      = errors.New("unknown cipher algorithm")
	ErrUnknownMAC         = errors.New("unknown MAC algorithm")
)

func checkSFTPAlgos(keyExchanges, ciphers, macs []string, forServer bool) error {
	for _, kex := range keyExchanges {
		if !validKeyExchanges.IsAlgoValid(kex, forServer) {
			return fmt.Errorf("%w %q", ErrUnknownKeyExchange, kex)
		}
	}

	for _, ciph := range ciphers {
		if !validCiphers.IsAlgoValid(ciph, forServer) {
			return fmt.Errorf("%w %q", ErrUnknownCipher, ciph)
		}
	}

	for _, mac := range macs {
		if !validMACs.IsAlgoValid(mac, forServer) {
			return fmt.Errorf("%w %q", ErrUnknownMAC, mac)
		}
	}

	return nil
}
