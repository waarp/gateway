package config

import (
	"encoding/base64"
	"fmt"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
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
	ServerPassword string `json:"serverPassword,omitempty"`

	// Specifies whether the partner uses TLS or not. Useless for servers.
	//nolint:tagliatelle // FIXME cannot be changed for compatibility reasons
	IsTLS bool `json:"isTLS,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`
}

// ValidPartner checks if the configuration is valid for a R66 partner.
func (c *R66ProtoConfig) ValidPartner() error {
	if len(c.ServerLogin) == 0 {
		return fmt.Errorf("missing partner login: %w", errInvalidProtoConfig)
	}

	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing partner password: %w", errInvalidProtoConfig)
	}

	pwd := r66.CryptPass([]byte(c.ServerPassword))
	c.ServerPassword = base64.StdEncoding.EncodeToString(pwd)

	return nil
}

// ValidServer checks if the configuration is valid for a R66 server.
func (c *R66ProtoConfig) ValidServer() error {
	if len(c.ServerPassword) == 0 {
		return fmt.Errorf("missing server password: %w", errInvalidProtoConfig)
	}

	pwd, err := utils.AESCrypt(c.ServerPassword)
	if err != nil {
		return fmt.Errorf("failed to crypt server password: %w", err)
	}

	c.ServerPassword = pwd

	return nil
}

// CertRequired returns whether, according to the configuration, a certificate
// is required for the R66 agent. Always returns false since the gateway does
// not implement R66 at the moment.
func (c *R66ProtoConfig) CertRequired() bool {
	return false
}
