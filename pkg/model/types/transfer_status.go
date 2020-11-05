package types

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

// this function has been commented out because it was unused. might be useful
// later
// func (t TransferStatus) isValid() bool {
// 	return t == StatusPlanned ||
// 		t == StatusTransfer ||
// 		t == StatusDone ||
// 		t == StatusError
// }

// ValidateStatusForTransfer returns whether the transfer value is valid for a
// model.Transfer entry.
func ValidateStatusForTransfer(t TransferStatus) bool {
	return t == StatusPlanned || t == StatusRunning || t == StatusPaused ||
		t == StatusInterrupted || t == StatusError
}

// ValidateStatusForHistory returns whether the transfer value is valid for a
// model.TransferHistory entry.
func ValidateStatusForHistory(t TransferStatus) bool {
	return t == StatusDone || t == StatusCancelled
}
