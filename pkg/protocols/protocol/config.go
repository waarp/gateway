package protocol

type (
	// ServerConfig is the JSON representation of a local server configuration.
	// The gateway should be able to unmarshal the JSON configuration of a server
	// into the underlying struct.
	// An empty JSON object should be a valid configuration.
	ServerConfig interface {
		// ValidServer checks whether the server configuration is valid, and
		// returns an error if it is not.
		ValidServer() error
	}

	// PartnerConfig is the JSON representation of a remote partner configuration.
	// The gateway should be able to unmarshal the JSON configuration of a partner
	// into the underlying struct.
	// An empty JSON object should be a valid configuration.
	PartnerConfig interface {
		// ValidPartner checks whether the partner configuration is valid, and
		// returns an error if it is not.
		ValidPartner() error
	}

	// ClientConfig is the JSON representation of a local client configuration.
	// The gateway should be able to unmarshal the JSON configuration of a client
	// into the underlying struct.
	// An empty JSON object should be a valid configuration.
	ClientConfig interface {
		// ValidClient checks whether the client configuration is valid, and
		// returns an error if it is not.
		ValidClient() error
	}
)
