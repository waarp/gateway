package config

import (
	"fmt"

	"code.waarp.fr/waarp-r66/r66"
)

func init() {
	ProtoConfigs["r66"] = func() ProtoConfig { return new(R66ProtoConfig) }
}

// R66ProtoConfig represents the configuration of a R66 agent.
type R66ProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`
	// The login used by the remote agent for server authentication.
	ServerLogin string `json:"serverLogin,omitempty"`
	// The server's password for server authentication.
	ServerPassword []byte `json:"serverPassword,omitempty"`
}

// ValidPartner checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidPartner() error {
	if len(c.ServerLogin) == 0 {
		return fmt.Errorf("missing partner login")
	}
	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing partner password")
	}
	c.ServerPassword = r66.CryptPass(c.ServerPassword)
	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() (err error) {
	if len(c.ServerLogin) != 0 {
		return fmt.Errorf("unknown JSON field 'serverLogin'")
	}
	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing server password")
	}
	c.ServerPassword = r66.CryptPass(c.ServerPassword)
	return nil
}
