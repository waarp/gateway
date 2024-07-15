package api

// InClient is the JSON object representing a local client in POST requests
// made to the gateway's REST server.
type InClient struct {
	Name         Nullable[string] `json:"name,omitempty"`
	Protocol     Nullable[string] `json:"protocol,omitempty"`
	Disabled     bool             `json:"disabled,omitempty"`
	LocalAddress Nullable[string] `json:"localAddress,omitempty"`
	ProtoConfig  UpdateObject     `json:"protoConfig,omitempty"`
}

// OutClient is the JSON object representing a local client in responses to GET
// requests made to the gateway's REST server.
type OutClient struct {
	Name         string         `json:"name"`
	Enabled      bool           `json:"enabled"`
	Protocol     string         `json:"protocol"`
	LocalAddress string         `json:"localAddress,omitempty"`
	ProtoConfig  map[string]any `json:"protoConfig"`
}
