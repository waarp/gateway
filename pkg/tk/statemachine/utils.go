package statemachine

// CanTransitionTo is a utility function which can be used when creating a
// MachineModel. It generates a map[State]struct with the given states as key.
// This map represents a list of possible transitions for the machine from a
// given state.
func CanTransitionTo(vals ...State) map[State]struct{} {
	m := map[State]struct{}{}
	for _, val := range vals {
		m[val] = struct{}{}
	}

	return m
}

// IsFinalState is a utility function which can be used when creating a
// MachineModel. It generates an empty map[State]struct. This map represents a
// state in which a machine no longer has any transition (i.e. a final state).
func IsFinalState() map[State]struct{} {
	return map[State]struct{}{}
}
