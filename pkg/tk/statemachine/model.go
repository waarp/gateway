package statemachine

// MachineModel is the struct representing a state machine model. Once a model
// is defined, it can be instantiated with the New function.
type MachineModel struct {
	Initial  State
	StateMap StateMap
}

// New returns a new instance of the state machine.
func (m *MachineModel) New() *Machine {
	return &Machine{
		current: m.Initial,
		states:  m.StateMap,
	}
}
