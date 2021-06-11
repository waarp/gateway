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

// PipelineSateMachine defines the different states and transitions of the
// pipeline's state machine.
var PipelineSateMachine = MachineModel{
	initial: "init",
	stateMap: StateMap{
		"init":            canTransitionTo("pre-tasks", "error"),
		"pre-tasks":       canTransitionTo("pre-tasks done", "error"),
		"pre-tasks done":  canTransitionTo("start data", "error"),
		"start data":      canTransitionTo("reading", "writing", "error"),
		"reading":         canTransitionTo("end data", "error"),
		"writing":         canTransitionTo("end data", "error"),
		"end data":        canTransitionTo("close", "error"),
		"close":           canTransitionTo("move", "error"),
		"move":            canTransitionTo("data ended", "error"),
		"data ended":      canTransitionTo("post-tasks", "error"),
		"post-tasks":      canTransitionTo("post-tasks done", "error"),
		"post-tasks done": canTransitionTo("end transfer", "error"),
		"end transfer":    canTransitionTo("all done", "error"),
		"all done":        isFinalState(),
		"error":           canTransitionTo("in error"),
		"in error":        isFinalState(),
	},
}
