package pesit

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const defaultMinTLSVersion = protoutils.TLSv12

var errSupportedTLSVersions = fmt.Errorf("supported TLS versions: %s",
	strings.Join([]string{
		protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13,
	}, ", "))

type ServerConfigTLS struct {
	// The standard Pesit options.
	ServerConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion string `json:"minTLSVersion"`
}

func (s *ServerConfigTLS) ValidServer() error {
	if s.MinTLSVersion == "" {
		s.MinTLSVersion = defaultMinTLSVersion
	}

	if protoutils.ParseTLSVersion(s.MinTLSVersion) == 0 {
		return fmt.Errorf("invalid TLS version %q: %w", s.MinTLSVersion, errSupportedTLSVersions)
	}

	return s.ServerConfig.ValidServer()
}

type PartnerConfigTLS struct {
	// The standard Pesit options.
	PartnerConfig

	// MinTLSVersion specifies the minimum TLS version allowed to communicate
	// with this partner. The accepted values are "v1.0", "v1.1", "v1.2", and
	// "v1.3". The default is "v1.2".
	MinTLSVersion string `json:"minTLSVersion"`
}

func (p *PartnerConfigTLS) ValidPartner() error {
	if p.MinTLSVersion == "" {
		p.MinTLSVersion = defaultMinTLSVersion
	}

	if protoutils.ParseTLSVersion(p.MinTLSVersion) == 0 {
		return fmt.Errorf("invalid TLS version %q: %w", p.MinTLSVersion, errSupportedTLSVersions)
	}

	return p.PartnerConfig.ValidPartner()
}

type ClientConfigTLS struct {
	// The standard Pesit options.
	ClientConfig
}

func (c *ClientConfigTLS) ValidClient() error {
	return c.ClientConfig.ValidClient()
}
