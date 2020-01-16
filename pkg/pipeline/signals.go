// Package pipeline regroups all the types and interfaces used in transfer pipelines.
package pipeline

// Signal is the type representing messages which can be sent to an executor
// during a transfer.
type Signal byte

const (
	// Shutdown is the signal sent to all transfer executors when the gateway is
	// shut down.
	Shutdown Signal = iota

	// Pause is the signal sent to a transfer executor when the current transfer
	// has been paused.
	Pause

	// Cancel is the signal sent to a transfer executor when the current transfer
	// has been cancelled.
	Cancel
)
