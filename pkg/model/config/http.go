//nolint:dupl //identical to https.go for now, keep separate for future-proofing
package config

//nolint:gochecknoinits // init is used by design
func init() {
	ProtoConfigs["http"] = &Constructor{
		Server:  func() ServerProtoConfig { return new(HTTPServerProtoConfig) },
		Partner: func() PartnerProtoConfig { return new(HTTPPartnerProtoConfig) },
		Client:  func() ClientProtoConfig { return new(HTTPClientProtoConfig) },
	}
}

// HTTPServerProtoConfig represents the configuration of a local HTTP server.
type HTTPServerProtoConfig struct{}

func (h *HTTPServerProtoConfig) ValidServer() error { return nil }

// HTTPPartnerProtoConfig represents the configuration of a remote HTTP partner.
type HTTPPartnerProtoConfig struct{}

func (h *HTTPPartnerProtoConfig) ValidPartner() error { return nil }

// HTTPClientProtoConfig represents the configuration of a local HTTP client.
type HTTPClientProtoConfig struct{}

func (h *HTTPClientProtoConfig) ValidClient() error { return nil }
