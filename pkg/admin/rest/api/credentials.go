package api

// InCred is the JSON representation of an authentication credential in POST
// requests made to the REST interface.
type InCred struct {
	Name   Nullable[string] `json:"name,omitempty"`
	Type   Nullable[string] `json:"type,omitempty"`
	Value  Nullable[string] `json:"value,omitempty"`
	Value2 Nullable[string] `json:"value2,omitempty"`
}
