package config

func init() {
	ProtoConfigs["http"] = func() ProtoConfig { return new(HTTPProtoConfig) }
	ProtoConfigs["https"] = func() ProtoConfig { return new(HTTPProtoConfig) }
}

// HTTPProtoConfig represents the configuration of an HTTP or HTTPS agent.
type HTTPProtoConfig struct{}

// ValidServer checks if the configuration is valid for a local HTTP server.
func (h *HTTPProtoConfig) ValidServer() error {
	return nil
}

// ValidPartner checks if the configuration is valid for an HTTP partner.
func (h *HTTPProtoConfig) ValidPartner() error {
	return nil
}
