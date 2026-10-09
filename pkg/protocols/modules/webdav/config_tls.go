package webdav

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

type ServerConfigTLS struct {
	ServerConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this server. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (s *ServerConfigTLS) ValidConf() error {
	return s.ServerConfig.ValidConf()
}

type PartnerConfigTLS struct {
	PartnerConfig

	// MinTLSVersion specifies the minimum TLS version allowed with this partner.
	// The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The default
	// is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this partner. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (p *PartnerConfigTLS) ValidConf() error {
	return p.PartnerConfig.ValidConf()
}

type ClientConfigTLS struct {
	ClientConfig

	// MinTLSVersion specifies the minimum TLS version allowed with this client.
	// The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The default
	// is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting using this client. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (c *ClientConfigTLS) ValidConf() error {
	return c.ClientConfig.ValidConf()
}
