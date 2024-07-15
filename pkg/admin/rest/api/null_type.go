package api

import (
	"encoding/json"
)

// Nullable represents a nullable JSON type. Can be used instead of pointers
// which can cause panics when handled improperly.
type Nullable[T any] struct {
	Value T    `json:"-"` // we have to add tags to stop the linter complaining
	Valid bool `json:"-"` // even though the struct implements json.(Un)Marshaler
}

func (n *Nullable[T]) UnmarshalJSON(bytes []byte) error {
	var val *T
	if err := json.Unmarshal(bytes, &val); err != nil {
		return err //nolint:wrapcheck //no need to wrap error here
	}

	if val != nil {
		n.Value = *val
		n.Valid = true
	}

	return nil
}

func (n *Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(n.Value) //nolint:wrapcheck //no need to wrap error here
}
