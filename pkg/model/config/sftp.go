package config

import (
	"fmt"
)

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs["sftp"] = func() ProtoConfig { return new(SftpProtoConfig) }
}

//nolint:gochecknoglobals // global var is used by design
var (
	SFTPValidKeyExchanges = Algos{
		{Name: "curve25519-sha256@libssh.org", ValidFor: Both},
		{Name: "ecdh-sha2-nistp256", ValidFor: Both},
		{Name: "ecdh-sha2-nistp384", ValidFor: Both},
		{Name: "ecdh-sha2-nistp521", ValidFor: Both},
		{Name: "diffie-hellman-group-exchange-sha256", ValidFor: OnlyClient},
		{Name: "diffie-hellman-group1-sha1", ValidFor: Both},  // Deprecated: uses SHA-1.
		{Name: "diffie-hellman-group14-sha1", ValidFor: Both}, // Deprecated: uses SHA-1.
	}
	SFTPValidCiphers = Algos{
		{Name: "aes128-gcm@openssh.com", ValidFor: Both},
		{Name: "chacha20-poly1305@openssh.com", ValidFor: Both},
		{Name: "aes128-ctr", ValidFor: Both},
		{Name: "aes192-ctr", ValidFor: Both},
		{Name: "aes256-ctr", ValidFor: Both},
		{Name: "arcfour256", ValidFor: Both, DisabledByDefault: true},
		{Name: "arcfour128", ValidFor: Both, DisabledByDefault: true},
		{Name: "arcfour", ValidFor: Both, DisabledByDefault: true},
		{Name: "aes128-cbc", ValidFor: Both, DisabledByDefault: true},
		{Name: "3des-cbc", ValidFor: Both, DisabledByDefault: true},
	}
	SFTPValidMACs = Algos{
		{Name: "hmac-sha2-256-etm@openssh.com", ValidFor: Both},
		{Name: "hmac-sha2-256", ValidFor: Both},
		{Name: "hmac-sha1", ValidFor: Both},    // Deprecated: uses SHA-1.
		{Name: "hmac-sha1-96", ValidFor: Both}, // Deprecated: uses SHA-1.
	}
)

// SftpProtoConfig represents the configuration of an SFTP agent.
type SftpProtoConfig struct {
	KeyExchanges                 []string `json:"keyExchanges,omitempty"`
	Ciphers                      []string `json:"ciphers,omitempty"`
	MACs                         []string `json:"macs,omitempty"`
	DisableClientConcurrentReads bool     `json:"disableClientConcurrentReads,omitempty"`
	UseStat                      bool     `json:"useStat,omitempty"`
}

func (c *SftpProtoConfig) valid(forServer bool) error {
	for _, kex := range c.KeyExchanges {
		if !SFTPValidKeyExchanges.isAlgoValid(kex, forServer) {
			return fmt.Errorf("unknown key exchange algorithm %q: %w", kex, errInvalidProtoConfig)
		}
	}

	for _, ciph := range c.Ciphers {
		if !SFTPValidCiphers.isAlgoValid(ciph, forServer) {
			return fmt.Errorf("unknown cipher algorithm %q: %w", ciph, errInvalidProtoConfig)
		}
	}

	for _, mac := range c.MACs {
		if !SFTPValidMACs.isAlgoValid(mac, forServer) {
			return fmt.Errorf("unknown MAC algorithm %q: %w", mac, errInvalidProtoConfig)
		}
	}

	return nil
}

// ValidPartner checks if the configuration is valid for an SFTP partner.
func (c *SftpProtoConfig) ValidPartner() error {
	return c.valid(false)
}

// ValidServer checks if the configuration is valid for a local SFTP server.
func (c *SftpProtoConfig) ValidServer() error {
	return c.valid(true)
}
