// Package types contains all the special types used by the database models.
package types

// TransferStatus represents the valid statuses for a transfer entry.
type TransferStatus string

const (
	// StatusPlanned is the state of a client transfer before it begins.
	StatusPlanned TransferStatus = "PLANNED"

	// StatusAvailable is the state of a server transfer before it begins.
	StatusAvailable TransferStatus = "AVAILABLE"

	// StatusRunning is the state of a transfer when it is running.
	StatusRunning TransferStatus = "RUNNING"

	// StatusInterrupted is the state of a transfer when interrupted unexpectedly.
	StatusInterrupted TransferStatus = "INTERRUPTED"

	// StatusPaused is the state of a transfer when paused by a user.
	StatusPaused TransferStatus = "PAUSED"

	// StatusCancelled is the state of a transfer when canceled by a user.
	StatusCancelled TransferStatus = "CANCELLED"

	// StatusDone is the state of a transfer when finished without error.
	StatusDone TransferStatus = "DONE"

	// StatusError is the state of a transfer when interrupted by an error.
	StatusError TransferStatus = "ERROR"
)

// StatusFromString takes a string and returns the corresponding TransferStatus.
// If the string does not match any status, the returned boolean will be false.
func StatusFromString(str string) (TransferStatus, bool) {
	switch str {
	case string(StatusPlanned):
		return StatusPlanned, true
	case string(StatusAvailable):
		return StatusAvailable, true
	case string(StatusRunning):
		return StatusRunning, true
	case string(StatusInterrupted):
		return StatusInterrupted, true
	case string(StatusPaused):
		return StatusPaused, true
	case string(StatusCancelled):
		return StatusCancelled, true
	case string(StatusDone):
		return StatusDone, true
	case string(StatusError):
		return StatusError, true
	default:
		return "", false
	}
}

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
	return t == StatusPlanned || t == StatusAvailable || t == StatusRunning ||
		t == StatusPaused || t == StatusInterrupted || t == StatusError
}

// ValidateStatusForHistory returns whether the transfer value is valid for a
// model.TransferHistory entry.
func ValidateStatusForHistory(t TransferStatus) bool {
	return t == StatusDone || t == StatusCancelled
}
