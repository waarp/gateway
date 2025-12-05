package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type GetCloudRespObject struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type,omitempty" yaml:"type,omitempty"`
	Key     string            `json:"key,omitempty" yaml:"key,omitempty"`
	Options map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

type PostCloudReqObject struct {
	Name    string      `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string      `json:"type,omitempty" yaml:"type,omitempty"`
	Key     string      `json:"key,omitempty" yaml:"key,omitempty"`
	Secret  string      `json:"secret,omitempty" yaml:"secret,omitempty"`
	Options CloudConfig `json:"options,omitempty" yaml:"options,omitempty"`
}

type PatchCloudReqObject struct {
	Name    string           `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string           `json:"type,omitempty" yaml:"type,omitempty"`
	Key     Nullable[string] `json:"key,omitzero" yaml:"key,omitempty"`
	Secret  Nullable[string] `json:"secret,omitzero" yaml:"secret,omitempty"`
	Options CloudConfig      `json:"options,omitempty" yaml:"options,omitempty"`
}

var ErrInvalidCloudConfigValue = errors.New("invalid cloud config value")

type CloudConfig map[string]string

func (c *CloudConfig) UnmarshalJSON(b []byte) error {
	data := map[string]any{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	*c = make(CloudConfig)

	for k, v := range data {
		switch val := v.(type) {
		case string:
			(*c)[k] = val
		case bool:
			(*c)[k] = strconv.FormatBool(val)
		case float64:
			(*c)[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case nil:
		default:
			return fmt.Errorf(`%w: "%v"`, ErrInvalidCloudConfigValue, v)
		}
	}

	return nil
}
