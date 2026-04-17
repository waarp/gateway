package webdav

type ServerConfig struct{}

func (s *ServerConfig) ValidConf() error { return nil }

type PartnerConfig struct{}

func (p *PartnerConfig) ValidConf() error { return nil }

type ClientConfig struct{}

func (c *ClientConfig) ValidConf() error { return nil }
