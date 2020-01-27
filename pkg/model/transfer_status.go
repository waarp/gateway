package model

// TransferStatus represents the valid statuses for a transfer entry.
type TransferStatus string

const (
	// StatusPlanned is the state of a transfer before it begins
	StatusPlanned TransferStatus = "PLANNED"

	// StatusPreTasks is the state of a transfer while it's running the pre tasks.
	StatusPreTasks TransferStatus = "PRE TASKS"

	// StatusTransfer is the state of a transfer while running
	StatusTransfer TransferStatus = "TRANSFER"

	// StatusPostTasks is the state of a transfer while it's running the post tasks.
	StatusPostTasks TransferStatus = "POST TASKS"

	// StatusErrorTasks is the state of a transfer while it's running the error tasks.
	StatusErrorTasks TransferStatus = "ERROR TASKS"

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

// this function has been commented out because it was unused. might be useful
// later
// func (t TransferStatus) isValid() bool {
// 	return t == StatusPlanned ||
// 		t == StatusTransfer ||
// 		t == StatusDone ||
// 		t == StatusError
// }

func validateStatusForTransfer(t TransferStatus) bool {
	return t == StatusPlanned || t == StatusTransfer || t == StatusPreTasks ||
		t == StatusPostTasks || t == StatusErrorTasks || t == StatusPaused
}

func validateStatusForHistory(t TransferStatus) bool {
	return t == StatusDone || t == StatusError || t == StatusCancelled
}
