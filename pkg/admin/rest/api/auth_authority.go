package api

type InAuthority struct {
	Name           Nullable[string] `json:"name,omitempty"`
	Type           Nullable[string] `json:"type,omitempty"`
	PublicIdentity Nullable[string] `json:"publicIdentity,omitempty"`
	ValidHosts     []string         `json:"validHosts,omitempty"`
}

type OutAuthority struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	PublicIdentity string   `json:"publicIdentity"`
	ValidHosts     []string `json:"validHosts"`
}
