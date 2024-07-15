package api

type GetCloudRespObject struct {
	Name    string         `json:"name"`
	Type    string         `json:"type,omitempty"`
	Key     string         `json:"key,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type PostCloudReqObject struct {
	Name    string         `json:"name,omitempty"`
	Type    string         `json:"type,omitempty"`
	Key     string         `json:"key,omitempty"`
	Secret  string         `json:"secret,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type PatchCloudReqObject struct {
	Name    string           `json:"name,omitempty"`
	Type    string           `json:"type,omitempty"`
	Key     Nullable[string] `json:"key,omitempty"`
	Secret  Nullable[string] `json:"secret,omitempty"`
	Options UpdateObject     `json:"options,omitempty"`
}
