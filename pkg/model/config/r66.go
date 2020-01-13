package config

import (
	"fmt"
)

// R66ProtoConfig represents the configuration of a R66 agent.
type R66ProtoConfig struct{}

// ValidClient checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidClient() error {
	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() error {
	return fmt.Errorf("protocol unsuported")
}
