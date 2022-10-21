// Package state defines the structure specifying the state of a gateway service.
package state

import (
	"sync"
)

// State handles a service global state. It associates a code with a reason.
// The reason contains a message explaining why the service is in this state.
type State struct {
	code   StateCode
	reason string
	mutex  sync.RWMutex
}

// Get returns the state code and the associated reason.
func (s *State) Get() (StateCode, string) {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	return s.code, s.reason
}

// Set allows one to change the StateCode of a service and the associated
// reason.
func (s *State) Set(code StateCode, reason string) {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	s.code = code
	s.reason = reason
}

// StateCode represents the state of a service.
// FIXME: Should implement json.Marshaler and json.Unmarshaler.
type StateCode uint8

const (
	// Offline is the state used to indicate that a service is stopped because
	// it has been disabled or stopped by the administrator. If it is not
	// working because of any error, the error state must be used.
	Offline StateCode = iota

	// Starting means the service is in its starting phase.
	Starting

	// Running means the service is alive and functions normally.
	Running

	// ShuttingDown means that the service has been asked to exit.
	ShuttingDown

	// Error means that the service is not properly functioning.
	Error
)

// Name returns the human representation is StateCode as a string.
// FIXME: Should be String().
func (s StateCode) Name() string {
	switch s {
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case ShuttingDown:
		return "Shutting down"
	case Error:
		return "Error"
	default:
		return "Offline"
	}
}
