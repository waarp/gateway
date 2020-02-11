package model

// TransferStatus represents the valid statuses for a transfer entry.
type TransferStatus string

const (
	// StatusPlanned is the state of a transfer before it begins.
	StatusPlanned TransferStatus = "PLANNED"

	// StatusRunning is the state of a transfer when it is running.
	StatusRunning TransferStatus = "RUNNING"

	// StatusInterrupted is the state of a transfer when interrupted unexpectedly.
	StatusInterrupted TransferStatus = "INTERRUPTED"

	// StatusPaused is the state of a transfer when paused by a user.
	StatusPaused TransferStatus = "PAUSED"

	// StatusCancelled is the state of a transfer when canceled by a user.
	StatusCancelled TransferStatus = "CANCELLED"

	// StatusDone is the state of a transfer when finished without error
	StatusDone TransferStatus = "DONE"

	// StatusError is the state of a transfer when interrupted by an error
	StatusError TransferStatus = "ERROR"
)

// TransferStep represents the different steps of a transfer.
type TransferStep string

const (
	// StepPreTasks is the state of a transfer while it's running the pre tasks.
	StepPreTasks TransferStep = "PRE TASKS"

	// StepData is the state of a transfer while transferring the file data.
	StepData TransferStep = "DATA"

	// StepPostTasks is the state of a transfer while it's running the post tasks.
	StepPostTasks TransferStep = "POST TASKS"

	// StepErrorTasks is the state of a transfer while it's running the error tasks.
	StepErrorTasks TransferStep = "ERROR TASKS"
)

// this function has been commented out because it was unused. might be useful
// later
// func (t TransferStatus) isValid() bool {
// 	return t == StatusPlanned ||
// 		t == StatusTransfer ||
// 		t == StatusDone ||
// 		t == StatusError
// }

func validateStatusForTransfer(t TransferStatus) bool {
	return t == StatusPlanned || t == StatusRunning || t == StatusPaused ||
		t == StatusInterrupted
}

func validateStatusForHistory(t TransferStatus) bool {
	return t == StatusDone || t == StatusError || t == StatusCancelled
}
