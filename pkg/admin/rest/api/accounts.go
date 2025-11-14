package api

// InRemoteAccount is the JSON representation of a remote account in POST requests
// made to the REST interface.
type InRemoteAccount struct {
	Login    Nullable[string] `json:"login,omitzero" yaml:"login,omitempty"`
	Password Nullable[string] `json:"password,omitzero" yaml:"password,omitempty"`
}

// OutRemoteAccount is the JSON representation of a remote account in responses
// sent by the REST interface.
type OutRemoteAccount struct {
	Login           string          `json:"login" yaml:"login"`
	Credentials     []string        `json:"credentials" yaml:"credentials"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules" yaml:"authorizedRules"`
}

// InLocalAccount is the JSON representation of a local account in POST requests
// made to the REST interface.
type InLocalAccount struct {
	Login       Nullable[string] `json:"login,omitzero" yaml:"login,omitempty"`
	Password    Nullable[string] `json:"password,omitzero" yaml:"password,omitempty"`
	IPAddresses List[string]     `json:"ipAddresses,omitzero" yaml:"ipAddresses,omitempty"`
}

// OutLocalAccount is the JSON representation of a local account in responses
// sent by the REST interface.
type OutLocalAccount struct {
	Login           string          `json:"login" yaml:"login"`
	Credentials     []string        `json:"credentials" yaml:"credentials"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules" yaml:"authorizedRules"`
	IPAddresses     []string        `json:"ipAddresses,omitempty" yaml:"ipAddresses,omitempty"`
}
