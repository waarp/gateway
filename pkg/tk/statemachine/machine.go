// Package statemachine contains an engine for creating and using simple state
// machines.
package statemachine

import (
	"sync"
)

// State represents a state for a state machine.
type State string

// StateMap is a map associating all the state machine's valid states along with
// the allowed transitions for each one of them.
type StateMap map[State]map[State]struct{}

// Machine is an instance of state machine. Transitions can be made using the
// Transition function.
type Machine struct {
	states  StateMap
	current State
	mutex   sync.RWMutex
}

// Current returns the current state of the state machine.
func (m *Machine) Current() State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.current
}

// Transition transitions the state machine to the given state. If the state is
// unknown or if the transition is not allowed, an error is returned.
func (m *Machine) Transition(newState State) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	validStates := m.states[m.current]

	_, ok := validStates[newState]
	if !ok {
		if _, ok := m.states[newState]; !ok {
			return &unknownStateError{newState}
		}

		return &transitionError{m.current, newState}
	}

	m.current = newState

	return nil
}

// HasEnded returns whether the state machine has reached a final state (from
// which no transition is possible) or not.
func (m *Machine) HasEnded() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	nextStates := m.states[m.current]

	return len(nextStates) == 0
}
