//nolint:dupl //identical to http_config.go for now, keep separate for future-proofing
package http

// httpsServerConfig represents the configuration of a local HTTP server.
type httpsServerConfig struct{}

func (h *httpsServerConfig) ValidServer() error { return nil }

// httpsPartnerConfig represents the configuration of a remote HTTP partner.
type httpsPartnerConfig struct{}

func (h *httpsPartnerConfig) ValidPartner() error { return nil }

// httpsClientConfig represents the configuration of a local HTTP client.
type httpsClientConfig struct{}

func (h *httpsClientConfig) ValidClient() error { return nil }
