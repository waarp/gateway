package main

import (
	"encoding/json"
	"fmt"
)

type confVal string

func (c confVal) MarshalJSON() ([]byte, error) {
	str := string(c)
	if !json.Valid([]byte(c)) {
		str = fmt.Sprintf(`"%s"`, c)
	}

	var val interface{}
	if err := json.Unmarshal([]byte(str), &val); err != nil {
		return nil, fmt.Errorf("cannot parse json: %w", err)
	}

	rv, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize value to JSON: %w", err)
	}

	return rv, nil
}
