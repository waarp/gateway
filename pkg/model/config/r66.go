package config

import "fmt"

func init() {
	ProtoConfigs["r66"] = func() ProtoConfig { return new(R66ProtoConfig) }
}

// R66ProtoConfig represents the configuration of a R66 agent.
type R66ProtoConfig struct {
	BlockSize      uint32 `json:"blockSize,omitempty"`
	ServerPassword []byte `json:"serverPassword,omitempty"`
}

// ValidPartner checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidPartner() error {
	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() error {
	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing server password")
	}
	return nil
}
