package api

// InAccount is the JSON representation of a local/remote account in POST requests
// made to the REST interface.
type InAccount struct {
	Login    Nullable[string] `json:"login,omitempty"`
	Password Nullable[string] `json:"password,omitempty"`
}

// OutAccount is the JSON representation of a local/remote account in responses
// sent by the REST interface.
type OutAccount struct {
	Login           string          `json:"login"`
	Credentials     []string        `json:"credentials"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}
