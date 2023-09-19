package utils

import (
	"errors"
	"sync"
)

var (
	ErrAlreadyRunning = errors.New("the service is already running")
	ErrNotRunning     = errors.New("the service is not running")
)

// State handles a service global state. It associates a code with a reason.
// The reason contains a message explaining why the service is in this state.
type State struct {
	code   StateCode
	reason string
	mutex  sync.RWMutex
}

func NewState(code StateCode, reason string) State {
	return State{
		code:   code,
		reason: reason,
	}
}

// Get returns the state code and the associated reason.
func (s *State) Get() (StateCode, string) {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	return s.code, s.reason
}

func (s *State) IsRunning() bool {
	code, _ := s.Get()

	return code == StateRunning
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
type StateCode uint8

const (
	// StateOffline is the state used to indicate that a service is stopped because
	// it has been disabled or stopped by the administrator. If it is not
	// working because of any error, the error state must be used.
	StateOffline StateCode = iota

	// StateRunning means the service is alive and functions normally.
	StateRunning

	// StateError means that the service is not properly functioning.
	StateError
)

func (s StateCode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// String returns the human representation is StateCode as a string.
func (s StateCode) String() string {
	switch s {
	case StateOffline:
		return "Offline"
	case StateRunning:
		return "Running"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}
