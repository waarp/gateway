package model

// Signal is the type representing messages which can be sent to an executor
// during a transfer.
type Signal byte

const (
	// SignalShutdown is the signal sent to all transfer executors when the gateway is
	// shut down.
	SignalShutdown Signal = iota

	// SignalPause is the signal sent to a transfer executor when the current transfer
	// has been paused.
	SignalPause

	// SignalCancel is the signal sent to a transfer executor when the current transfer
	// has been cancelled.
	SignalCancel
)
