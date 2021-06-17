package pipeline

import (
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// errLimitReached is the error returned when a counter cannot be incremented
// because the maximum limit has been reached.
var errLimitReached = types.NewTransferError(types.TeExceededLimit, "transfer limit reached")

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
