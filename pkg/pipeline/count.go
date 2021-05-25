package pipeline

import (
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// errLimitReached is the error returned when a counter cannot be incremented
// because the maximum limit has been reached.
var errLimitReached = types.NewTransferError(types.TeExceededLimit, "transfer limit reached")

// TransferInterrupter is the interface stored in the RunningTransfers map. The
// functions exposed by this interface can be used to interrupt running transfers.
type TransferInterrupter interface {
	// Pause pauses the transfer. It can be resumed later on command.
	Pause()

	// Interrupt stops the transfer because of a service shutdown. Transfer will
	// be resumed automatically when the service restarts.
	Interrupt()

	// Cancel cancels the transfer. Transfer will be moved to history and thus
	// cannot be resumed.
	Cancel()
}

var (
	// TransferInCount counts the current and maximum number of concurrent incoming
	// transfers. A limit of 0 means no limit.
	TransferInCount = &count{}

	// TransferOutCount counts the current and maximum number of concurrent outgoing
	// transfers. A limit of 0 means no limit.
	TransferOutCount = &count{}
)

// count is a thread-safe counter with a maximum limit check included.
type count struct {
	count uint64
	limit uint64
	mux   sync.Mutex
}

// SetLimit sets the maximum limit of the counter.
func (c *count) SetLimit(l uint64) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.limit = l
}

func (c *count) Add() (added bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.limit == 0 {
		return true
	}

	newCount := c.count + 1
	if newCount > c.limit {
		return false
	}
	c.count = newCount
	return true
}

func (c *count) Sub() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.count--
}

func (c *count) GetAvailable() uint64 {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.limit == 0 {
		return 0
	}
	return c.limit - c.count
}
