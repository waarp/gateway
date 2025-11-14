package api

type GetCloudRespObject struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type,omitempty" yaml:"type,omitempty"`
	Key     string            `json:"key,omitempty" yaml:"key,omitempty"`
	Options map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

type PostCloudReqObject struct {
	Name    string            `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string            `json:"type,omitempty" yaml:"type,omitempty"`
	Key     string            `json:"key,omitempty" yaml:"key,omitempty"`
	Secret  string            `json:"secret,omitempty" yaml:"secret,omitempty"`
	Options map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

type PatchCloudReqObject struct {
	Name    string               `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string               `json:"type,omitempty" yaml:"type,omitempty"`
	Key     Nullable[string]     `json:"key,omitzero" yaml:"key,omitempty"`
	Secret  Nullable[string]     `json:"secret,omitzero" yaml:"secret,omitempty"`
	Options UpdateObject[string] `json:"options,omitempty" yaml:"options,omitempty"`
}
