package models

// Status is the status of the service
type Status struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// Statuses maps a service name to its state
type Statuses map[string]Status
