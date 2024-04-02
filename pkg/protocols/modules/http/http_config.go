//nolint:dupl //identical to https_config.go for now, keep separate for future-proofing
package http

// serverConfig represents the configuration of a local HTTP server.
type serverConfig struct{}

func (h *serverConfig) ValidServer() error { return nil }

// partnerConfig represents the configuration of a remote HTTP partner.
type partnerConfig struct{}

func (h *partnerConfig) ValidPartner() error { return nil }

// clientConfig represents the configuration of a local HTTP client.
type clientConfig struct{}

func (h *clientConfig) ValidClient() error { return nil }
