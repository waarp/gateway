package config

import (
	"fmt"
)

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs["sftp"] = &ConfigMaker{
		Server:  func() ServerProtoConfig { return new(SftpServerProtoConfig) },
		Partner: func() PartnerProtoConfig { return new(SftpPartnerProtoConfig) },
		Client:  func() ClientProtoConfig { return new(SftpClientProtoConfig) },
	}
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

func checkSFTPAlgos(keyExchanges, ciphers, macs []string, forServer bool) error {
	for _, kex := range keyExchanges {
		if !SFTPValidKeyExchanges.isAlgoValid(kex, forServer) {
			return fmt.Errorf("unknown key exchange algorithm %q: %w", kex, errInvalidProtoConfig)
		}
	}

	for _, ciph := range ciphers {
		if !SFTPValidCiphers.isAlgoValid(ciph, forServer) {
			return fmt.Errorf("unknown cipher algorithm %q: %w", ciph, errInvalidProtoConfig)
		}
	}

	for _, mac := range macs {
		if !SFTPValidMACs.isAlgoValid(mac, forServer) {
			return fmt.Errorf("unknown MAC algorithm %q: %w", mac, errInvalidProtoConfig)
		}
	}

	return nil
}

// SftpServerProtoConfig represents the configuration of a local SFTP server.
type SftpServerProtoConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *SftpServerProtoConfig) ValidServer() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, true)
}

// SftpPartnerProtoConfig represents the configuration of a remote SFTP partner.
type SftpPartnerProtoConfig struct {
	KeyExchanges                 []string `json:"keyExchanges,omitempty"`
	Ciphers                      []string `json:"ciphers,omitempty"`
	MACs                         []string `json:"macs,omitempty"`
	DisableClientConcurrentReads bool     `json:"disableClientConcurrentReads,omitempty"`
	UseStat                      bool     `json:"useStat,omitempty"`
}

func (s *SftpPartnerProtoConfig) ValidPartner() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}

// SftpClientProtoConfig represents the configuration of a local SFTP client.
type SftpClientProtoConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *SftpClientProtoConfig) ValidClient() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}
