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
	states        StateMap
	current, last State
	mutex         sync.RWMutex
}

// Current returns the current state of the state machine.
func (m *Machine) Current() State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.current
}

// Last returns the last state of the state machine.
func (m *Machine) Last() State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.last
}

// Transition transitions the state machine to the given state. If the state is
// unknown or if the transition is not allowed, an error is returned.
func (m *Machine) Transition(newState State) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.transition(newState)
}

func (m *Machine) isValidTransition(newState State) error {
	validStates := m.states[m.current]

	if _, isValid := validStates[newState]; !isValid {
		if _, exists := m.states[newState]; !exists {
			return &unknownStateError{newState}
		}

		return &transitionError{m.current, newState}
	}

	return nil
}

func (m *Machine) doTransition(newState State) {
	m.last = m.current
	m.current = newState
}

func (m *Machine) transition(newState State) error {
	if err := m.isValidTransition(newState); err != nil {
		return err
	}

	m.doTransition(newState)

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
