package api

// InPartner is the JSON representation of a remote agent in requests
// made to the REST interface.
type InPartner struct {
	Name        Nullable[string]  `json:"name,omitzero" yaml:"name,omitempty"`
	Protocol    Nullable[string]  `json:"protocol,omitzero" yaml:"protocol,omitempty"`
	Address     Nullable[string]  `json:"address,omitzero" yaml:"address,omitempty"`
	ProtoConfig UpdateObject[any] `json:"protoConfig,omitempty" yaml:"protoConfig,omitempty"`
}

// OutPartner is the JSON representation of a remote partner in responses sent
// by the REST interface.
type OutPartner struct {
	Name            string          `json:"name" yaml:"name"`
	Protocol        string          `json:"protocol" yaml:"protocol"`
	Address         string          `json:"address" yaml:"address"`
	Credentials     []string        `json:"credentials" yaml:"credentials"`
	ProtoConfig     map[string]any  `json:"protoConfig" yaml:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules" yaml:"authorizedRules"`
}
