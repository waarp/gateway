package api

import "encoding/json"

type List[T any] []T

func (l *List[T]) UnmarshalJSON(bytes []byte) error {
	var val []T
	if err := json.Unmarshal(bytes, &val); err != nil {
		return err //nolint:wrapcheck //no need to wrap error here
	}

	if val != nil {
		*l = val
	}

	return nil
}
