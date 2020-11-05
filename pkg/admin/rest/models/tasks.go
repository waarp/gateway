package models

import "encoding/json"

// Task is the JSON representation of a rule task in requests made to
// the REST interface.
type Task struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}
