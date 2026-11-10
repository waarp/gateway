package api

import (
	"encoding/json"
	"reflect"
)

// Nullable represents a nullable JSON type. Can be used instead of pointers
// which can cause panics when handled improperly.
type Nullable[T any] struct {
	Value T    `json:"-"` // we have to add tags to stop the linter complaining
	Valid bool `json:"-"` // even though the struct implements json.(Un)Marshaler
}

func NewNullable[T any](value T) Nullable[T] {
	if isNil(value) {
		return Nullable[T]{}
	}

	return Nullable[T]{
		Value: value,
		Valid: true,
	}
}

func (n Nullable[T]) IsZero() bool { return !n.Valid }

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

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(n.Value) //nolint:wrapcheck //no need to wrap error here
}

func (n *Nullable[T]) UnmarshalYAML(f func(any) error) error {
	if err := f(&n.Value); err != nil {
		return err
	}

	if !isNil(n.Value) {
		n.Valid = true
	}

	return nil
}

func (n Nullable[T]) MarshalYAML() (any, error) {
	if !n.Valid {
		//nolint:nilnil //nil, nil is correct here according to the interface's spec
		return nil, nil
	}

	return n.Value, nil
}

func isNil[T any](value T) (isNil bool) {
	defer func() {
		if recover() != nil {
			isNil = false
		}
	}()

	rvalue := reflect.ValueOf(value)

	return !rvalue.IsValid() || rvalue.IsNil()
}
