package api

import "encoding/json"

type UpdateObject map[string]any

func (u *UpdateObject) UnmarshalJSON(bytes []byte) error {
	var v map[string]any
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	if v != nil {
		*u = v
	}

	return nil
}
