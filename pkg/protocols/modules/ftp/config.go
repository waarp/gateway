package ftp

type ServerConfig struct {
	// DisablePassiveMode indicates if passive mode should be disabled.
	// By default, passive mode is enabled.
	DisablePassiveMode bool `json:"disablePassiveMode"`

	// DisableActiveMode indicates if active mode should be disabled.
	// By default, active mode is enabled.
	DisableActiveMode bool `json:"disableActiveMode"`

	// PassiveModeMinPort indicates the starting port number of the port range
	// which the server is allowed to use for data transfer in passive mode.
	// By default, any free port is allowed.
	PassiveModeMinPort uint16 `json:"passiveModeMinPort,omitempty"`

	// PassiveModeMaxPort indicates the ending port number of the port range
	// which the server is allowed to use for data transfer in passive mode.
	// By default, any free port is allowed.
	PassiveModeMaxPort uint16 `json:"passiveModeMaxPort,omitempty"`
}

func (s *ServerConfig) ValidServer() error {
	if s.DisablePassiveMode {
		s.PassiveModeMinPort = 0
		s.PassiveModeMaxPort = 0
	}

	return nil
}

type ClientConfig struct {
	// EnableActiveMode indicates if active mode should be enabled. By default,
	// active mode is disabled.
	EnableActiveMode bool `json:"enableActiveMode"`

	// ActiveModeAddress indicates the local IP address of the client in
	// active mode for data transfers. The default value is 0.0.0.0.
	ActiveModeAddress string `json:"activeModeAddress,omitempty"`

	// ActiveModeMinPort indicates the starting port number of the port range
	// which the client is allowed to use for data transfer in active mode.
	// By default, any free port is allowed.
	ActiveModeMinPort uint16 `json:"activeModeMinPort,omitempty"`

	// ActiveModeMaxPort indicates the ending port number of the port range
	// which the client is allowed to use for data transfer in active mode.
	// By default, any free port is allowed.
	ActiveModeMaxPort uint16 `json:"activeModeMaxPort,omitempty"`
}

func (c *ClientConfig) ValidClient() error {
	if !c.EnableActiveMode {
		c.ActiveModeMinPort = 0
		c.ActiveModeMaxPort = 0
	}

	return nil
}

type PartnerConfig struct {
	// DisableActiveMode indicates if active mode should be disabled when
	// communicating with this FTP partner specifically (assuming the client
	// used to make the connection has enabled active mode). By default,
	// active mode is enabled if the client allows it.
	DisableActiveMode bool `json:"disableActiveMode"`

	// DisableEPSV indicates if EPSV (or Extended Passive Mode) should be
	// disabled when communicating with this FTP partner, since some servers
	// do not support that command. By default, EPSV is enabled.
	//
	//nolint:tagliatelle //EPSV is an accronym, keep in caps
	DisableEPSV bool `json:"disableEPSV,omitempty"`
}

func (p *PartnerConfig) ValidPartner() error {
	return nil
}
