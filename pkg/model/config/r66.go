package config

import (
	"fmt"
)

func init() {
	ProtoConfigs["r66"] = func() ProtoConfig { return new(SftpProtoConfig) }
}

// R66ProtoConfig represents the configuration of a R66 agent.
type R66ProtoConfig struct{}

// ValidPartner checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidPartner() error {
	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() error {
	return fmt.Errorf("protocol unsuported")
}
