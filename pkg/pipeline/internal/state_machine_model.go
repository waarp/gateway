package internal

// MachineModel is the struct representing a state machine model. Once a model
// is defined, it can be instantiated with the New function.
type MachineModel struct {
	initial  string
	stateMap StateMap
}

// New returns a new instance of the state machine.
func (m *MachineModel) New() *Machine {
	return &Machine{
		current: m.initial,
		states:  m.stateMap,
	}
}
