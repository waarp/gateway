package api

// InRemoteAccount is the JSON representation of a remote account in POST requests
// made to the REST interface.
type InRemoteAccount struct {
	Login    Nullable[string] `json:"login,omitempty"`
	Password Nullable[string] `json:"password,omitempty"`
}

// OutRemoteAccount is the JSON representation of a remote account in responses
// sent by the REST interface.
type OutRemoteAccount struct {
	Login           string          `json:"login"`
	Credentials     []string        `json:"credentials"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}

// InLocalAccount is the JSON representation of a local account in POST requests
// made to the REST interface.
type InLocalAccount struct {
	Login       Nullable[string] `json:"login,omitempty"`
	Password    Nullable[string] `json:"password,omitempty"`
	IPAddresses List[string]     `json:"ipAddresses,omitempty"`
}

// OutLocalAccount is the JSON representation of a local account in responses
// sent by the REST interface.
type OutLocalAccount struct {
	Login           string          `json:"login"`
	Credentials     []string        `json:"credentials"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
	IPAddresses     []string        `json:"ipAddresses,omitempty"`
}
