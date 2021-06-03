package internal

import (
	"fmt"
	"sync"
)

type StateMap map[string]map[string]struct{}

type Machine struct {
	states  StateMap
	current string
	mutex   sync.RWMutex
}

func (m *Machine) Current() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.current
}

func (m *Machine) Transition(newState string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	validStates := m.states[m.current]
	_, ok := validStates[newState]
	if !ok {
		if _, ok := m.states[newState]; !ok {
			return fmt.Errorf("unknown state '%s'", newState)
		}
		return fmt.Errorf("invalid state transition from %s to %s", m.current, newState)
	}
	m.current = newState
	return nil
}

func (m *Machine) HasEnded() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	nextStates := m.states[m.current]
	return len(nextStates) == 0
}

func canTransitionTo(vals ...string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, val := range vals {
		m[val] = struct{}{}
	}
	return m
}

func isFinalState() map[string]struct{} {
	return map[string]struct{}{}
}
