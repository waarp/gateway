// Package api contains all the struct models describing the various JSON
// objects used by the REST API.
package api

// InUser is the JSON representation of a user account in requests made to the
// REST interface.
type InUser struct {
	Username Nullable[string] `json:"username,omitzero" yaml:"username,omitempty"`
	Password Nullable[string] `json:"password,omitzero" yaml:"password,omitempty"`
	Perms    Perms            `json:"perms,omitzero" yaml:"perms,omitempty"`
}

// OutUser is the JSON representation of a user account in responses sent by
// the REST interface.
type OutUser struct {
	Username string `json:"username" yaml:"username"`
	Perms    Perms  `json:"perms" yaml:"perms"`
}

// Perms is a struct regrouping a user's permissions into different categories.
type Perms struct {
	Transfers      string `json:"transfers" yaml:"transfers"`
	Servers        string `json:"servers" yaml:"servers"`
	Partners       string `json:"partners" yaml:"partners"`
	Rules          string `json:"rules" yaml:"rules"`
	Users          string `json:"users" yaml:"users"`
	Administration string `json:"administration" yaml:"administration"`
}
