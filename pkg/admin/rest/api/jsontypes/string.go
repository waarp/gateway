// Package jsontypes provides custom types which can be used for JSON objects
// marshaling and unmarshalling.
package jsontypes

import "encoding/json"

type NullString struct {
	String string
	Valid  bool
}

func NewNullString(s string) NullString {
	return NullString{String: s, Valid: true}
}

func (n *NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(n.String) //nolint:wrapcheck //wrapping here adds nothing
}

func (n *NullString) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err //nolint:wrapcheck //wrapping here adds nothing
	}

	if s != nil {
		n.Valid = true
		n.String = *s
	}

	return nil
}
