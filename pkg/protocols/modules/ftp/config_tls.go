package ftp

import (
	"errors"
	"fmt"
	"strings"

	ftplib "github.com/fclairamb/ftpserverlib"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const defaultMinTLSVersion = protoutils.TLSv12

var (
	errSupportedTLSVersions = fmt.Errorf("supported TLS versions: %s",
		strings.Join([]string{protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13}, ", "))
	errSupportedTLSRequirements = errors.New(`supported TLS requirements: "Optional", "Mandatory" & "Implicit"`)
)

type ServerConfigTLS struct {
	ServerConfig

	// TLSRequirement specifies the server's requirement for TLS usage. The
	// accepted values are "Optional" (for optional explicit TLS), "Mandatory"
	// (for mandatory explicit TLS), and "Implicit" (for implicit TLS).
	// The default is "Optional".
	TLSRequirement TLSRequirement `json:"tlsRequirement"`

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion string `json:"minTLSVersion"`
}

func (c *ServerConfigTLS) ValidServer() error {
	if c.TLSRequirement == "" {
		c.TLSRequirement = TLSOptional
	}

	if c.TLSRequirement.toLib() < 0 {
		return fmt.Errorf("invalid TLS requirement %q: %w", c.TLSRequirement, errSupportedTLSRequirements)
	}

	if c.MinTLSVersion == "" {
		c.MinTLSVersion = defaultMinTLSVersion
	}

	if protoutils.ParseTLSVersion(c.MinTLSVersion) == 0 {
		return fmt.Errorf("invalid TLS version %q: %w", c.MinTLSVersion, errSupportedTLSVersions)
	}

	return c.ServerConfig.ValidServer()
}

type ClientConfigTLS struct {
	ClientConfig

	// MinTLSVersion specifies the minimum TLS version that the client should
	// allow. The accepted values are "1.0", "1.1", "1.2", and "1.3". The
	// default is "1.2".
	MinTLSVersion string `json:"minTLSVersion"`
}

func (c *ClientConfigTLS) ValidClient() error {
	if c.MinTLSVersion == "" {
		c.MinTLSVersion = defaultMinTLSVersion
	}

	if protoutils.ParseTLSVersion(c.MinTLSVersion) == 0 {
		return fmt.Errorf("invalid TLS version %q: %w", c.MinTLSVersion, errSupportedTLSVersions)
	}

	return c.ClientConfig.ValidClient()
}

type PartnerConfigTLS struct {
	PartnerConfig

	// UseImplicitTLS states whether this partner should be configured to use
	// implicit TLS over explicit TLS. In explicit TLS mode, the client first
	// opens a plain-text control connection, and then "upgrades" it to TLS
	// via the "AUTH TLS" command. In Implicit TLS mode, the client must open
	// a TLS control connection directly from the start, and the server will
	// refuse any non-TLS connections. By default, implicit TLS is used.
	UseImplicitTLS bool `json:"useImplicitTLS"`

	// MinTLSVersion specifies the minimum TLS version that the client should
	// allow. The accepted values are "1.0", "1.1", "1.2", and "1.3". The
	// default is "1.2".
	MinTLSVersion string `json:"minTLSVersion"`

	// DisableTLSSessionReuse states whether TLS session reuse should be
	// disabled. By default, TLS session are reused when opening the data
	// connection, but some servers do not support this feature.
	DisableTLSSessionReuse bool `json:"disableTLSSessionReuse"`
}

func (c *PartnerConfigTLS) ValidPartner() error {
	if c.MinTLSVersion == "" {
		c.MinTLSVersion = defaultMinTLSVersion
	}

	if protoutils.ParseTLSVersion(c.MinTLSVersion) == 0 {
		return fmt.Errorf("invalid TLS version %q: %w", c.MinTLSVersion, errSupportedTLSVersions)
	}

	return c.PartnerConfig.ValidPartner()
}

type TLSRequirement string

const (
	TLSOptional  TLSRequirement = "Optional"  // Optional explicit TLS
	TLSMandatory TLSRequirement = "Mandatory" // Mandatory explicit TLS
	TLSImplicit  TLSRequirement = "Implicit"  // Implicit TLS
)

func (req TLSRequirement) toLib() ftplib.TLSRequirement {
	switch req {
	case TLSOptional, "":
		return ftplib.ClearOrEncrypted
	case TLSMandatory:
		return ftplib.MandatoryEncryption
	case TLSImplicit:
		return ftplib.ImplicitEncryption
	default:
		return -1
	}
}
