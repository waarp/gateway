package api

// InAccount is the JSON representation of a local/remote account in requests
// made to the REST interface.
type InAccount struct {
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
}

// OutAccount is the JSON representation of a local/remote account in responses
// sent by the REST interface.
type OutAccount struct {
	Login           string           `json:"login"`
	AuthorizedRules *AuthorizedRules `json:"authorizedRules"`
}
