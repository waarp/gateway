package config

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

const (
	ProtocolR66    = "r66"
	ProtocolR66TLS = "r66-tls"
)

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs[ProtocolR66] = func() ProtoConfig { return new(R66ProtoConfig) }
	ProtoConfigs[ProtocolR66TLS] = func() ProtoConfig { return new(R66ProtoConfig) }
}

// R66ProtoConfig represents the configuration of a R66 agent.
type R66ProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the remote agent for server authentication.
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	ServerPassword string `json:"serverPassword,omitempty"`

	// Specifies whether the partner uses TLS or not. Useless for servers.
	// Deprecated: use the r66-tls protocol instead.
	//nolint:tagliatelle // FIXME cannot be changed for compatibility reasons
	IsTLS *bool `json:"isTLS,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

// ValidPartner checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidPartner() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.ServerLogin) == 0 {
		return fmt.Errorf("missing partner login: %w", errInvalidProtoConfig)
	}

	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing partner password: %w", errInvalidProtoConfig)
	}

	if utils.IsHash(c.ServerPassword) {
		return nil // password already hashed
	}

	pwd := utils.R66Hash(c.ServerPassword)

	hashed, err := utils.HashPassword(database.BcryptRounds, pwd)
	if err != nil {
		return fmt.Errorf("failed to hash server password: %w", err)
	}

	c.ServerPassword = hashed

	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing server password: %w", errInvalidProtoConfig)
	}

	pwd, err := utils.AESCrypt(database.GCM, c.ServerPassword)
	if err != nil {
		return fmt.Errorf("failed to crypt server password: %w", err)
	}

	c.ServerPassword = pwd

	return nil
}
