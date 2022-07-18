package config

import (
	"fmt"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

const ProtocolR66TLS = "r66-tls"

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs[ProtocolR66TLS] = &ConfigMaker{
		Server:  func() ServerProtoConfig { return new(R66TLSServerProtoConfig) },
		Partner: func() PartnerProtoConfig { return new(R66TLSPartnerProtoConfig) },
		Client:  func() ClientProtoConfig { return new(R66TLSClientProtoConfig) },
	}
}

// R66TLSServerProtoConfig represents the configuration of a local R66-TLS server.
type R66TLSServerProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication.
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	ServerPassword string `json:"serverPassword,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *R66TLSServerProtoConfig) ValidServer() error {
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

// R66TLSPartnerProtoConfig represents the configuration of a remote R66-TLS partner.
type R66TLSPartnerProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication.
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	ServerPassword string `json:"serverPassword,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

//nolint:dupl //it's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *R66TLSPartnerProtoConfig) ValidPartner() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing partner password: %w", errInvalidProtoConfig)
	}

	if _, err := bcrypt.Cost([]byte(c.ServerPassword)); err == nil {
		return nil // password already hashed
	}

	pwd := r66.CryptPass([]byte(c.ServerPassword))

	hashed, err := utils.HashPassword(database.BcryptRounds, string(pwd))
	if err != nil {
		return fmt.Errorf("failed to hash server password: %w", err)
	}

	c.ServerPassword = hashed

	return nil
}

// R66TLSClientProtoConfig represents the configuration of a local R66-TLS client.
type R66TLSClientProtoConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *R66TLSClientProtoConfig) ValidClient() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	return nil
}
