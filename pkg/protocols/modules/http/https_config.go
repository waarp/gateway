//nolint:dupl //identical to http_config.go for now, keep separate for future-proofing
package http

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

// httpsServerConfig represents the configuration of a local HTTP server.
type httpsServerConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (h *httpsServerConfig) ValidServer() error { return nil }

// httpsPartnerConfig represents the configuration of a remote HTTP partner.
type httpsPartnerConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (h *httpsPartnerConfig) ValidPartner() error { return nil }

// httpsClientConfig represents the configuration of a local HTTP client.
type httpsClientConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (h *httpsClientConfig) ValidClient() error { return nil }
