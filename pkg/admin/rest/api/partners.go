package api

// InPartner is the JSON representation of a remote agent in requests
// made to the REST interface.
type InPartner struct {
	Name        *string        `json:"name,omitempty"`
	Protocol    *string        `json:"protocol,omitempty"`
	Address     *string        `json:"address,omitempty"`
	ProtoConfig map[string]any `json:"protoConfig,omitempty"`
}

// OutPartner is the JSON representation of a remote partner in responses sent
// by the REST interface.
type OutPartner struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	Address         string          `json:"address"`
	ProtoConfig     map[string]any  `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}
