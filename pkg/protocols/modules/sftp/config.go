package sftp

// serverConfig represents the configuration of a local SFTP server.
type serverConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *serverConfig) ValidServer() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, true)
}

// partnerConfig represents the configuration of a remote SFTP partner.
type partnerConfig struct {
	KeyExchanges                 []string `json:"keyExchanges,omitempty"`
	Ciphers                      []string `json:"ciphers,omitempty"`
	MACs                         []string `json:"macs,omitempty"`
	DisableClientConcurrentReads bool     `json:"disableClientConcurrentReads,omitempty"`
	UseStat                      bool     `json:"useStat,omitempty"`
}

func (s *partnerConfig) ValidPartner() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}

// clientConfig represents the configuration of a local SFTP client.
type clientConfig struct {
	KeyExchanges []string `json:"keyExchanges,omitempty"`
	Ciphers      []string `json:"ciphers,omitempty"`
	MACs         []string `json:"macs,omitempty"`
}

func (s *clientConfig) ValidClient() error {
	return checkSFTPAlgos(s.KeyExchanges, s.Ciphers, s.MACs, false)
}
