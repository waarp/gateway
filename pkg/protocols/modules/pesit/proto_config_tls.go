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
}

func (s *ServerConfigTLS) ValidConf() error {
	return s.ServerConfig.ValidConf()
}

type PartnerConfigTLS struct {
	// The standard Pesit options.
	PartnerConfig

	// MinTLSVersion specifies the minimum TLS version allowed to communicate
	// with this partner. The accepted values are "v1.0", "v1.1", "v1.2", and
	// "v1.3". The default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (p *PartnerConfigTLS) ValidConf() error {
	return p.PartnerConfig.ValidConf()
}

type ClientConfigTLS struct {
	// The standard Pesit options.
	ClientConfig
}

func (c *ClientConfigTLS) ValidConf() error {
	return c.ClientConfig.ValidConf()
}
