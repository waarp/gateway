package config

import (
	"fmt"
)

func init() {
	ProtoConfigs["sftp"] = func() ProtoConfig { return new(SftpProtoConfig) }
}

var (
	validKeyExchange = map[string]interface{}{
		"diffie-hellman-group1-sha1":   nil,
		"diffie-hellman-group14-sha1":  nil,
		"ecdh-sha2-nistp256":           nil,
		"ecdh-sha2-nistp384":           nil,
		"ecdh-sha2-nistp521":           nil,
		"curve25519-sha256@libssh.org": nil,
	}
	validCipher = map[string]interface{}{
		"aes128-gcm@openssh.com":        nil,
		"aes128-ctr":                    nil,
		"aes192-ctr":                    nil,
		"aes256-ctr":                    nil,
		"chacha20-poly1305@openssh.com": nil,
	}
	validMACs = map[string]interface{}{
		"hmac-sha2-256-etm@openssh.com": nil,
		"hmac-sha2-256":                 nil,
		"hmac-sha1":                     nil,
		"hmac-sha1-96":                  nil,
	}
)

// SftpProtoConfig represents the configuration of an SFTP agent.
type SftpProtoConfig struct {
	Port         uint16   `json:"port"`
	Address      string   `json:"address"`
	KeyExchanges []string `json:"keyExchanges"`
	Ciphers      []string `json:"ciphers"`
	MACs         []string `json:"macs"`
}

// ValidPartner checks if the configuration is valid for an SFTP partner.
func (c *SftpProtoConfig) ValidPartner() error {
	if c.Port == 0 {
		return fmt.Errorf("sftp port must be specified")
	}
	if c.Address == "" {
		return fmt.Errorf("sftp address cannot be empty")
	}

	for _, k := range c.KeyExchanges {
		if _, ok := validKeyExchange[k]; !ok {
			return fmt.Errorf("unknown key exchange algorithm '%s'", k)
		}
	}
	for _, c := range c.Ciphers {
		if _, ok := validCipher[c]; !ok {
			return fmt.Errorf("unknown cipher algorithm '%s'", c)
		}
	}
	for _, m := range c.MACs {
		if _, ok := validMACs[m]; !ok {
			return fmt.Errorf("unknown MAC algorithm '%s'", m)
		}
	}

	return nil
}

// ValidServer checks if the configuration is valid for a local SFTP server.
func (c *SftpProtoConfig) ValidServer() error {
	return c.ValidPartner()
}
