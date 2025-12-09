package api

// InCred is the JSON representation of an authentication credential in POST
// requests made to the REST interface.
type InCred struct {
	Name   Nullable[string] `json:"name,omitzero" yaml:"name,omitempty"`
	Type   Nullable[string] `json:"type,omitzero" yaml:"type,omitempty"`
	Value  Nullable[string] `json:"value,omitzero" yaml:"value,omitempty"`
	Value2 Nullable[string] `json:"value2,omitzero" yaml:"value2,omitempty"`
}

type OutCred struct {
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
	Type   string `json:"type,omitempty" yaml:"type,omitempty"`
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Value2 string `json:"value2,omitempty" yaml:"value2,omitempty"`
}
