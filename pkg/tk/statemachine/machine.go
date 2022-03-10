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

// DeferTransition checks whether the state machine can transition to the given
// state but defer executing the transition to the caller. If the transition is
// possible, this function returns a function which can be used by the caller to
// execute the transition. Once DeferTransition has returned with no error, no
// other transition will be possible until the returned function is called.
func (m *Machine) DeferTransition(newState State) (do func(), err error) {
	m.mutex.Lock()

	if err := m.isValidTransition(newState); err != nil {
		m.mutex.Unlock()

		return nil, err
	}

	once := sync.Once{}

	return func() {
		once.Do(func() {
			defer m.mutex.Unlock()

			m.doTransition(newState)
		})
	}, nil
}

func (m *Machine) isValidTransition(newState State) error {
	validStates := m.states[m.current]

	_, ok := validStates[newState]
	if !ok {
		if _, ok := m.states[newState]; !ok {
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
