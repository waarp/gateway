package api

// Task is the JSON representation of a rule task in requests made to
// the REST interface.
type Task struct {
	Type string            `json:"type" yaml:"type"`
	Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
}
