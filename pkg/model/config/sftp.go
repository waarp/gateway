package config

import (
	"fmt"
)

// SftpProtoConfig represents the configuration of an SFTP agent.
type SftpProtoConfig struct {
	Port    uint16
	Address string
	Root    string
}

// ValidClient checks if the configuration is valid for an SFTP partner.
func (c *SftpProtoConfig) ValidClient() error {
	if c.Port == 0 {
		return fmt.Errorf("sftp port must be specified")
	}
	if c.Address == "" {
		return fmt.Errorf("sftp address cannot be empty")
	}
	return nil
}

// ValidServer checks if the configuration is valid for a local SFTP server.
func (c *SftpProtoConfig) ValidServer() error {
	if c.Port == 0 {
		return fmt.Errorf("sftp port must be specified")
	}
	if c.Address == "" {
		return fmt.Errorf("sftp address cannot be empty")
	}
	if c.Root == "" {
		return fmt.Errorf("sftp root cannot be empty")
	}
	return nil
}
