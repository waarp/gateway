//nolint:dupl //identical to http_config.go for now, keep separate for future-proofing
package http

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

// httpsServerConfig represents the configuration of a local HTTP server.
type httpsServerConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this server. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (h *httpsServerConfig) ValidConf() error { return nil }

// httpsPartnerConfig represents the configuration of a remote HTTP partner.
type httpsPartnerConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this partner. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (h *httpsPartnerConfig) ValidConf() error { return nil }

// httpsClientConfig represents the configuration of a local HTTP client.
type httpsClientConfig struct {
	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting using this client. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (h *httpsClientConfig) ValidConf() error { return nil }
