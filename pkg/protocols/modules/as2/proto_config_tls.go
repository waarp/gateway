package as2

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

type clientProtoConfigTLS struct {
	clientProtoConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting using this client. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (c *clientProtoConfigTLS) ValidConf() error {
	return c.clientProtoConfig.ValidConf()
}

type partnerProtoConfigTLS struct {
	partnerProtoConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this partner. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (p *partnerProtoConfigTLS) ValidConf() error {
	return p.partnerProtoConfig.ValidConf()
}

type serverProtoConfigTLS struct {
	serverProtoConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this server. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (s *serverProtoConfigTLS) ValidConf() error {
	return s.serverProtoConfig.ValidConf()
}
