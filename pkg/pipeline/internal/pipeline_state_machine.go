package internal

const (
	PipelineInitState          = "init"
	PipelineErrorState         = "error"
	PipelineInErrorState       = "in error"
	PipelinePreTasksState      = "pre-tasks"
	PipelinePreTasksDoneState  = "pre-tasks done"
	PipelineStartDataState     = "start data"
	PipelineReadState          = "reading"
	PipelineWriteState         = "writing"
	PipelineEndDataState       = "end data"
	PipelineCloseState         = "close"
	PipelineMoveState          = "move"
	PipelineDataEndState       = "data ended"
	PipelinePostTasksState     = "post-tasks"
	PipelinePostTasksDoneState = "post-tasks done"
	PipelineEndTransferState   = "end transfer"
	PipelineAllDoneState       = "all done"
)

// PipelineSateMachine defines the different states and transitions of the
// pipeline's state machine.
//nolint:gochecknoglobals // global var is used by design
var PipelineSateMachine = MachineModel{
	initial: PipelineInitState,
	stateMap: StateMap{
		PipelineInitState:          canTransitionTo(PipelinePreTasksState, PipelineErrorState),
		PipelinePreTasksState:      canTransitionTo(PipelinePreTasksDoneState, PipelineErrorState),
		PipelinePreTasksDoneState:  canTransitionTo(PipelineStartDataState, PipelineErrorState),
		PipelineStartDataState:     canTransitionTo(PipelineReadState, PipelineWriteState, PipelineErrorState),
		PipelineReadState:          canTransitionTo(PipelineEndDataState, PipelineErrorState),
		PipelineWriteState:         canTransitionTo(PipelineEndDataState, PipelineErrorState),
		PipelineEndDataState:       canTransitionTo(PipelineCloseState, PipelineErrorState),
		PipelineCloseState:         canTransitionTo(PipelineMoveState, PipelineErrorState),
		PipelineMoveState:          canTransitionTo(PipelineDataEndState, PipelineErrorState),
		PipelineDataEndState:       canTransitionTo(PipelinePostTasksState, PipelineErrorState),
		PipelinePostTasksState:     canTransitionTo(PipelinePostTasksDoneState, PipelineErrorState),
		PipelinePostTasksDoneState: canTransitionTo(PipelineEndTransferState, PipelineErrorState),
		PipelineEndTransferState:   canTransitionTo(PipelineAllDoneState, PipelineErrorState),
		PipelineAllDoneState:       isFinalState(),
		PipelineErrorState:         canTransitionTo(PipelineInErrorState),
		PipelineInErrorState:       isFinalState(),
	},
}
