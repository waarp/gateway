// Package api contains all the struct models describing the various JSON
// objects used by the REST API.
package api

// InUser is the JSON representation of a user account in requests made to the
// REST interface.
type InUser struct {
	Username *string `json:"username,omitempty"`
	Password []byte  `json:"password,omitempty"`
}

// OutUser is the JSON representation of a user account in responses sent by
// the REST interface.
type OutUser struct {
	Username string `json:"username"`
}
