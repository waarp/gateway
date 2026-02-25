package webdav

type ServerConfig struct{}

func (s *ServerConfig) ValidServer() error { return nil }

type PartnerConfig struct{}

func (p *PartnerConfig) ValidPartner() error { return nil }

type ClientConfig struct{}

func (c *ClientConfig) ValidClient() error { return nil }
