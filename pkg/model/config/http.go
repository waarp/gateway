package config

func init() {
	ProtoConfigs["http"] = func() ProtoConfig { return new(HTTPProtoConfig) }
}

// HTTPProtoConfig represents the configuration of an HTTP agent.
type HTTPProtoConfig struct {
	UseHTTPS bool `json:"useHTTPS,omitempty"`
}

// ValidServer checks if the configuration is valid for a local SFTP server.
func (h *HTTPProtoConfig) ValidServer() error {
	return nil
}

// ValidPartner checks if the configuration is valid for an HTTP partner.
func (h *HTTPProtoConfig) ValidPartner() error {
	return nil
}
