package model

import (
	"bytes"
	"encoding/json"
)

type Map[T any] map[string]T

func (m Map[T]) Map() map[string]T { return m }

func (m Map[T]) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return json.Marshal(map[string]T(m))
}

//nolint:wrapcheck //wrapping adds nothing here
func (m *Map[T]) UnmarshalJSON(b []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()

	mp := map[string]T{}

	if err := decoder.Decode(&mp); err != nil {
		return err
	}

	*m = mp

	return nil
}

type List[T any] []T

func (l List[T]) List() []T { return l }

func (l List[T]) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("[]"), nil
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return json.Marshal([]T(l))
}

//nolint:wrapcheck //wrapping adds nothing here
func (l *List[T]) UnmarshalJSON(b []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()

	var ls []T
	if err := decoder.Decode(&ls); err != nil {
		return err
	}

	*l = ls

	return nil
}
