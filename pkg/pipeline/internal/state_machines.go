package internal

type MachineModel struct {
	initial  string
	stateMap StateMap
}

func (m *MachineModel) New() *Machine {
	return &Machine{
		current: m.initial,
		states:  m.stateMap,
	}
}

var PipelineSateMachine = MachineModel{
	initial: "init",
	stateMap: StateMap{
		"init":            canTransitionTo("pre-tasks", "error"),
		"pre-tasks":       canTransitionTo("pre-tasks done", "error"),
		"pre-tasks done":  canTransitionTo("start data", "error"),
		"start data":      canTransitionTo("reading", "writing", "error"),
		"reading":         canTransitionTo("close", "error"),
		"writing":         canTransitionTo("close", "error"),
		"close":           canTransitionTo("move", "error"),
		"move":            canTransitionTo("end data", "error"),
		"end data":        canTransitionTo("post-tasks", "error"),
		"post-tasks":      canTransitionTo("post-tasks done", "error"),
		"post-tasks done": canTransitionTo("end transfer", "error"),
		"end transfer":    canTransitionTo("all done", "error"),
		"all done":        isFinalState(),
		"error":           canTransitionTo("in error"),
		"in error":        isFinalState(),
	},
}
