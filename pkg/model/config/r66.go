package config

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

const ProtocolR66 = "r66"

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs[ProtocolR66] = &Constructor{
		Server:  func() ServerProtoConfig { return new(R66ServerProtoConfig) },
		Partner: func() PartnerProtoConfig { return new(R66PartnerProtoConfig) },
		Client:  func() ClientProtoConfig { return new(R66ClientProtoConfig) },
	}
}

// R66ServerProtoConfig represents the configuration of a local R66 server.
type R66ServerProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication.
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

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ServerProtoConfig) ValidServer() error {
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

// R66PartnerProtoConfig represents the configuration of a remote R66 partner.
type R66PartnerProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication.
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	ServerPassword string `json:"serverPassword,omitempty"`

	// Specifies whether the partner uses TLS or not. Useless for servers.
	// Deprecated: use the r66-tls protocol instead.
	//nolint:tagliatelle // FIXME cannot be changed for compatibility reasons
	IsTLS *bool `json:"isTLS,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash *bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash *bool `json:"checkBlockHash,omitempty"`
}

// ValidPartner checks if the configuration is valid for a R66 partner.
//
//nolint:dupl //It's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *R66PartnerProtoConfig) ValidPartner() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
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

// R66ClientProtoConfig represents the configuration of a local R66 client.
type R66ClientProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

// ValidClient checks if the configuration is valid for an R66 client.
func (c *R66ClientProtoConfig) ValidClient() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	return nil
}
