package r66

import (
	"fmt"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// tlsServerConfig represents the configuration of a local R66-TLS server.
type tlsServerConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication (if different
	// from the server's name).
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *tlsServerConfig) ValidServer() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.ServerPassword) == 0 {
		return nil
	}

	pwd, err := utils.AESCrypt(database.GCM, c.ServerPassword)
	if err != nil {
		return fmt.Errorf("failed to crypt server password: %w", err)
	}

	c.ServerPassword = pwd

	return nil
}

// tlsPartnerConfig represents the configuration of a remote R66-TLS partner.
type tlsPartnerConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the partner for server authentication (if different
	// from the partner's name).
	ServerLogin string `json:"serverLogin,omitempty"`

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

//nolint:dupl //it's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *tlsPartnerConfig) ValidPartner() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.ServerPassword) == 0 {
		return nil
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

// tlsClientConfig represents the configuration of a local R66-TLS client.
type tlsClientConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *tlsClientConfig) ValidClient() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	return nil
}
