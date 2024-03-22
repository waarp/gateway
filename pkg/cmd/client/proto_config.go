package wg

import (
	"encoding/json"
	"fmt"
)

type confVal string

func (c confVal) MarshalJSON() ([]byte, error) {
	if !json.Valid([]byte(c)) {
		return []byte(fmt.Sprintf(`"%s"`, c)), nil
	}

	return []byte(c), nil
}
