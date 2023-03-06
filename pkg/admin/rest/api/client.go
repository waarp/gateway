package api

// InClient is the JSON object representing a local client in POST, PUT & PATCH
// requests made to the gateway's REST server.
type InClient struct {
	Name         string         `json:"name,omitempty"`
	Protocol     string         `json:"protocol,omitempty"`
	LocalAddress *string        `json:"localAddress,omitempty"`
	ProtoConfig  map[string]any `json:"protoConfig,omitempty"`
}

// OutClient is the JSON object representing a local client in responses to GET
// requests made to the gateway's REST server.
type OutClient struct {
	Name         string         `json:"name"`
	Protocol     string         `json:"protocol"`
	LocalAddress string         `json:"localAddress"`
	ProtoConfig  map[string]any `json:"protoConfig"`
}
