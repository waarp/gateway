package api

import "encoding/json"

type UpdateObject[T any] map[string]T

func (u *UpdateObject[T]) UnmarshalJSON(bytes []byte) error {
	var v map[string]T
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	if v != nil {
		*u = v
	}

	return nil
}
