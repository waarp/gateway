package api

// InFilewatcher represents the JSON body for creating or partially updating a
// filewatcher via the REST API. All fields are nullable so that a PATCH request
// only modifies the fields that are explicitly provided.
type InFilewatcher struct {
	Flow             Nullable[string]   `json:"flow,omitzero" yaml:"flow,omitempty"`
	Disabled         Nullable[bool]     `json:"disabled,omitzero" yaml:"disabled,omitempty"`
	Interval         Nullable[Duration] `json:"interval,omitzero" yaml:"interval,omitempty"`
	Pattern          Nullable[string]   `json:"pattern,omitzero" yaml:"pattern,omitempty"`
	NoDuplicateCheck Nullable[bool]     `json:"noDuplicateCheck,omitzero" yaml:"noDuplicateCheck,omitempty"`
	Partner          Nullable[string]   `json:"partner,omitzero" yaml:"partner,omitempty"`
	Account          Nullable[string]   `json:"account,omitzero" yaml:"account,omitempty"`
	Client           Nullable[string]   `json:"client,omitzero" yaml:"client,omitempty"`
	Rule             Nullable[string]   `json:"rule,omitzero" yaml:"rule,omitempty"`
}

// OutFilewatcher represents the JSON body returned for GET requests on filewatchers.
type OutFilewatcher struct {
	Flow             string   `json:"flow" yaml:"flow"`
	Disabled         bool     `json:"disabled" yaml:"disabled"`
	Interval         Duration `json:"interval" yaml:"interval"`
	Pattern          string   `json:"pattern" yaml:"pattern"`
	NoDuplicateCheck bool     `json:"noDuplicateCheck" yaml:"noDuplicateCheck"`
	Partner          string   `json:"partner" yaml:"partner"`
	Account          string   `json:"account" yaml:"account"`
	Client           string   `json:"client" yaml:"client"`
	Rule             string   `json:"rule" yaml:"rule"`
}
