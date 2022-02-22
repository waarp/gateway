package pipeline

import "code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"

const (
	stateInit          statemachine.State = "init"
	stateError         statemachine.State = "error"
	stateInError       statemachine.State = "in error"
	statePreTasks      statemachine.State = "pre-tasks"
	statePreTasksDone  statemachine.State = "pre-tasks done"
	stateReading       statemachine.State = "reading"
	stateWriting       statemachine.State = "writing"
	stateDataEnd       statemachine.State = "data end"
	stateDataEndDone   statemachine.State = "data ended"
	statePostTasks     statemachine.State = "post-tasks"
	statePostTasksDone statemachine.State = "post-tasks done"
	stateEndTransfer   statemachine.State = "end transfer"
	stateAllDone       statemachine.State = "all done"
)

// pipelineSateMachine defines the different states and transitions of the
// pipeline's state machine.
//nolint:gochecknoglobals // global var is used by design
var pipelineSateMachine = statemachine.MachineModel{
	Initial: stateInit,
	StateMap: statemachine.StateMap{
		stateInit:          statemachine.CanTransitionTo(statePreTasks, stateError),
		statePreTasks:      statemachine.CanTransitionTo(statePreTasksDone, stateError),
		statePreTasksDone:  statemachine.CanTransitionTo(stateReading, stateWriting, stateError),
		stateReading:       statemachine.CanTransitionTo(stateDataEnd, stateError),
		stateWriting:       statemachine.CanTransitionTo(stateDataEnd, stateError),
		stateDataEnd:       statemachine.CanTransitionTo(stateDataEndDone, stateError),
		stateDataEndDone:   statemachine.CanTransitionTo(statePostTasks, stateError),
		statePostTasks:     statemachine.CanTransitionTo(statePostTasksDone, stateError),
		statePostTasksDone: statemachine.CanTransitionTo(stateEndTransfer, stateError),
		stateEndTransfer:   statemachine.CanTransitionTo(stateAllDone, stateError),
		stateAllDone:       statemachine.IsFinalState(),
		stateError:         statemachine.CanTransitionTo(stateInError),
		stateInError:       statemachine.IsFinalState(),
	},
}
