package pipeline

// Trace is a struct which can be given to a pipeline when initializing it. The
// struct regroups a bunch of functions which, if set, will be called by the
// pipeline at various points of its execution. This allows the caller to trace
// the pipeline's progression through the calls to these functions. These
// functions can also be used to simulate errors during the pipeline's execution
// for testing purposes.
type Trace struct {
	// OnInit is called at the end of the pipeline's initialization.
	OnInit func() error
	// OnTransferStart is called at the very beginning of the transfer's execution.
	OnTransferStart func() error
	// OnPreTask is called after each pre-task. The task's rank is given as
	// argument.
	OnPreTask func(rank int) error
	// OnDataStart is called when the data transfer starts.
	OnDataStart func() error
	// OnRead is called each time data is read from the transfer stream. The
	// read offset is given as argument.
	OnRead func(off int64) error
	// OnWrite is called each time data is written to the transfer stream. The
	// write offset is given as argument.
	OnWrite func(off int64) error
	// OnClose is called when the transfer stream is closed.
	OnClose func() error
	// OnMove is called when the temp receive file is moved to its final
	// destination (only on receiver side)/
	OnMove func() error
	// OnPostTask is called after each post-task. The task's rank is given as
	// argument.
	OnPostTask func(rank int) error
	// OnFinalization is called during the transfer's finalization, when the
	// transfer ends normally.
	OnFinalization func() error
	// OnError is called when an error occurs at any point during the transfer.
	// The error in question is given as function parameter.
	OnError func(cause error)
	// OnErrorTask is called after each error-task. The task's rank is given as
	// argument.
	OnErrorTask func(rank int)
	// OnPause is called when the transfer is paused.
	OnPause func()
	// OnInterruption is called when the transfer is interrupted by a shutdown.
	OnInterruption func()
	// OnCancel is called when the transfer is canceled.
	OnCancel func()
	// OnTransferEnd is called when the transfer is finished, both normally or
	// if interrupted.
	OnTransferEnd func()
}
