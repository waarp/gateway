package api

type InAuthority struct {
	Name           Nullable[string] `json:"name,omitzero" yaml:"name,omitempty"`
	Type           Nullable[string] `json:"type,omitzero" yaml:"type,omitempty"`
	PublicIdentity Nullable[string] `json:"publicIdentity,omitzero" yaml:"publicIdentity,omitempty"`
	ValidHosts     []string         `json:"validHosts,omitempty" yaml:"validHosts,omitempty"`
}

type OutAuthority struct {
	Name           string   `json:"name" yaml:"name"`
	Type           string   `json:"type" yaml:"type"`
	PublicIdentity string   `json:"publicIdentity" yaml:"publicIdentity"`
	ValidHosts     []string `json:"validHosts" yaml:"validHosts"`
}
