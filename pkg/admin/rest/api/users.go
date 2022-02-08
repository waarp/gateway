// Package api contains all the struct models describing the various JSON
// objects used by the REST API.
package api

// InUser is the JSON representation of a user account in requests made to the
// REST interface.
type InUser struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Perms    *Perms  `json:"perms"`
}

// OutUser is the JSON representation of a user account in responses sent by
// the REST interface.
type OutUser struct {
	Username string `json:"username"`
	Perms    Perms  `json:"perms"`
}

// Perms is a struct regrouping a user's permissions into different categories.
type Perms struct {
	Transfers      string `json:"transfers"`
	Servers        string `json:"servers"`
	Partners       string `json:"partners"`
	Rules          string `json:"rules"`
	Users          string `json:"users"`
	Administration string `json:"administration"`
}
