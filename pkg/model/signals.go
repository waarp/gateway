package model

// Signal is the type representing messages which can be sent to an executor
// during a transfer.
type Signal byte

const (
	// SignalPause is the signal sent to a transfer executor when the current transfer
	// has been paused.
	SignalPause Signal = iota

	// SignalCancel is the signal sent to a transfer executor when the current transfer
	// has been cancelled.
	SignalCancel
)
