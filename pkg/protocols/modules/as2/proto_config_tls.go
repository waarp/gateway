package as2

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

type clientProtoConfigTLS struct {
	clientProtoConfig

	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (c *clientProtoConfigTLS) ValidConf() error {
	return c.clientProtoConfig.ValidConf()
}

type partnerProtoConfigTLS struct {
	partnerProtoConfig

	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (p *partnerProtoConfigTLS) ValidConf() error {
	return p.partnerProtoConfig.ValidConf()
}

type serverProtoConfigTLS struct {
	serverProtoConfig

	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (s *serverProtoConfigTLS) ValidConf() error {
	return s.serverProtoConfig.ValidConf()
}
