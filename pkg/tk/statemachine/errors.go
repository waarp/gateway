package statemachine

import "fmt"

type transitionError struct {
	current, new State
}

func (t *transitionError) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s", t.current, t.new)
}

type unknownStateError struct {
	state State
}

func (u *unknownStateError) Error() string {
	return fmt.Sprintf("unknown state '%s'", u.state)
}
