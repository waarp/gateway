package r66

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// serverConfig represents the configuration of a local R66 server.
type serverConfig struct {
	sharedServerConfig

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// Specifies whether the partner uses TLS or not. Useless for servers.
	// Deprecated: use the r66-tls protocol instead.
	IsTLS *bool `json:"isTLS,omitempty"`
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *serverConfig) ValidServer() error {
	if err := encryptServerPassword(&c.ServerPassword); err != nil {
		return err
	}

	return c.ValidShared()
}

// PartnerConfig represents the configuration of a remote R66 partner.
type PartnerConfig struct {
	sharedPartnerConfig

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// Specifies whether the partner uses TLS or not. Useless for servers.
	// Deprecated: use the r66-tls protocol instead.
	IsTLS *bool `json:"isTLS,omitempty"` //nolint:tagliatelle // FIXME cannot be changed for compatibility reasons
}

// ValidPartner checks if the configuration is valid for a R66 partner.
//
//nolint:dupl //It's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *PartnerConfig) ValidPartner() error {
	if c.ServerPassword == "" {
		return nil
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

// clientConfig represents the configuration of a local R66 client.
type clientConfig struct {
	sharedClientConfig
}

// ValidClient checks if the configuration is valid for an R66 client.
func (c *clientConfig) ValidClient() error {
	return c.ValidShared()
}
