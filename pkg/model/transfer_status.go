package model

// TransferStatus represents the valid statuses for a transfer entry.
type TransferStatus string

const (
	// StatusPlanned is the state of a transfer before it begins
	StatusPlanned TransferStatus = "PLANNED"

	// StatusTransfer is the state of a transfer while running
	StatusTransfer TransferStatus = "TRANSFER"

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
	return t == StatusPlanned || t == StatusTransfer
}

func validateStatusForHistory(t TransferStatus) bool {
	return t == StatusDone || t == StatusError
}
