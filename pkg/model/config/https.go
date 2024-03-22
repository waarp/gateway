//nolint:dupl //identical to https.go for now, keep separate for future-proofing
package config

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs["https"] = &Constructor{
		Server:  func() ServerProtoConfig { return new(HTTPSServerProtoConfig) },
		Partner: func() PartnerProtoConfig { return new(HTTPSPartnerProtoConfig) },
		Client:  func() ClientProtoConfig { return new(HTTPSClientProtoConfig) },
	}
}

// HTTPSServerProtoConfig represents the configuration of a local HTTP server.
type HTTPSServerProtoConfig struct{}

func (h *HTTPSServerProtoConfig) ValidServer() error { return nil }

// HTTPSPartnerProtoConfig represents the configuration of a remote HTTP partner.
type HTTPSPartnerProtoConfig struct{}

func (h *HTTPSPartnerProtoConfig) ValidPartner() error { return nil }

// HTTPSClientProtoConfig represents the configuration of a local HTTP client.
type HTTPSClientProtoConfig struct{}

func (h *HTTPSClientProtoConfig) ValidClient() error { return nil }
