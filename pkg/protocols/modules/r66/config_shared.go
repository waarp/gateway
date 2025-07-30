package r66

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"

type sharedServerConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the server for server authentication (if different
	// from the server's name).
	ServerLogin string `json:"serverLogin,omitempty"`

	// Specifies whether the server uses the legacy R66 certificate for TLS.
	// Useless if the server does not use TLS.
	// Deprecated: use model.Credential instead.
	UsesLegacyCertificate bool `json:"usesLegacyCertificate,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// Specifies the algorithms allowed for the final hash verification.
	FinalHashAlgos []string `json:"finalHashAlgos,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *sharedServerConfig) ValidShared() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if len(c.FinalHashAlgos) == 0 {
		c.FinalHashAlgos = []string{internal.HashSHA256}
	}

	return nil
}

type sharedPartnerConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// The login used by the partner for server authentication (if different
	// from the partner's name).
	ServerLogin string `json:"serverLogin,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash *bool `json:"noFinalHash,omitempty"`

	// The Hash algorithm used to check the validity of the files transferred.
	FinalHashAlgo string `json:"finalHashAlgo,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash *bool `json:"checkBlockHash,omitempty"`
}

func (c *sharedPartnerConfig) ValidShared() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if c.FinalHashAlgo == "" {
		c.FinalHashAlgo = internal.HashSHA256
	}

	return nil
}

type sharedClientConfig struct {
	// The block size for transfers. Optional, 65536 by default.
	BlockSize uint32 `json:"blockSize,omitempty"`

	// If true, the final hash verification will be disabled.
	NoFinalHash bool `json:"noFinalHash,omitempty"`

	// The Hash algorithm used to check the validity of the files transferred.
	FinalHashAlgo string `json:"finalHashAlgo,omitempty"`

	// If true, a hash check will be performed on each block during a transfer.
	CheckBlockHash bool `json:"checkBlockHash,omitempty"`
}

func (c *sharedClientConfig) ValidShared() error {
	if c.BlockSize == 0 {
		c.BlockSize = 65536
	}

	if c.FinalHashAlgo == "" {
		c.FinalHashAlgo = internal.HashSHA256
	}

	return nil
}
