package sftp

// ServerConfig represents the configuration of a local SFTP server.
type ServerConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *ServerConfig) ValidConf() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, true)
}

// PartnerConfig represents the configuration of a remote SFTP partner.
type PartnerConfig struct {
	KeyExchanges                 []string `json:"keyExchanges,omitempty"`
	Ciphers                      []string `json:"ciphers,omitempty"`
	MACs                         []string `json:"macs,omitempty"`
	DisableClientConcurrentReads bool     `json:"disableClientConcurrentReads,omitempty"`
	UseStat                      bool     `json:"useStat,omitempty"`
}

func (s *PartnerConfig) ValidConf() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}

// ClientConfig represents the configuration of a local SFTP client.
type ClientConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *ClientConfig) ValidConf() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}
