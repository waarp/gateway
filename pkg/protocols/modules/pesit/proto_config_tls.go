package pesit

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type ServerConfigTLS struct {
	// The standard Pesit options.
	ServerConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of accepted TLS cipher suites by name.
	// If empty, Go defaults are used. Use this to force specific suites for
	// legacy mainframe interoperability.
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

func (s *ServerConfigTLS) ValidServer() error {
	if _, err := resolveCipherSuites(s.CipherSuites); err != nil {
		return err
	}

	return s.ServerConfig.ValidServer()
}

type PartnerConfigTLS struct {
	// The standard Pesit options.
	PartnerConfig

	// MinTLSVersion specifies the minimum TLS version allowed to communicate
	// with this partner. The accepted values are "v1.0", "v1.1", "v1.2", and
	// "v1.3". The default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this partner. If empty, Go defaults are used.
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

func (p *PartnerConfigTLS) ValidPartner() error {
	if _, err := resolveCipherSuites(p.CipherSuites); err != nil {
		return err
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
