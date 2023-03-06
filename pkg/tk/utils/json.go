package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JSONConvert takes a source object and converts it to the destination object
// using JSON to map the object fields. Useful to convert between structs and
// maps (and vice-versa).
func JSONConvert(from, to any) error {
	var (
		buf     bytes.Buffer
		encoder = json.NewEncoder(&buf)
		decoder = json.NewDecoder(&buf)
	)

	decoder.DisallowUnknownFields()

	if err := encoder.Encode(from); err != nil {
		return fmt.Errorf("failed to re-encode the JSON object: %w", err)
	}

	if err := decoder.Decode(to); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}
