package api

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api/jsontypes"
)

type GetCloudRespObject struct {
	Name    string         `json:"name"`
	Type    string         `json:"type,omitempty"`
	Key     string         `json:"key,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type PostCloudReqObject struct {
	Name    string           `json:"name,omitempty"`
	Type    string           `json:"type,omitempty"`
	Key     string           `json:"key,omitempty"`
	Secret  string           `json:"secret,omitempty"`
	Options jsontypes.Object `json:"options,omitempty"`
}

type PatchCloudReqObject struct {
	Name    string               `json:"name,omitempty"`
	Type    string               `json:"type,omitempty"`
	Key     jsontypes.NullString `json:"key,omitempty"`
	Secret  jsontypes.NullString `json:"secret,omitempty"`
	Options jsontypes.Object     `json:"options,omitempty"`
}
