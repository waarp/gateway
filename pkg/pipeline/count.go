package pipeline

import (
	"fmt"
	"sync"
)

// ErrLimitReached is the error returned when a counter cannot be incremented
// because the maximum limit has been reached.
var ErrLimitReached = fmt.Errorf("transfer limit reached")

// TransferInCount counts the current and maximum number of concurrent incoming
// transfers. A limit of 0 means no limit.
var TransferInCount = &Count{}

// TransferOutCount counts the current and maximum number of concurrent outgoing
// transfers. A limit of 0 means no limit.
var TransferOutCount = &Count{}

// Count is a thread-safe counter with a maximum limit check included.
type Count struct {
	count uint64
	limit uint64
	mux   sync.Mutex
}

// SetLimit sets the maximum limit of the counter.
func (c *Count) SetLimit(l uint64) {
	defer c.mux.Unlock()
	c.mux.Lock()

	c.limit = l
}

// GetLimit sets the maximum limit of the counter.
func (c *Count) GetLimit() uint64 {
	defer c.mux.Unlock()
	c.mux.Lock()

	return c.limit
}

// Get returns the current value of the counter.
func (c *Count) Get() uint64 {
	defer c.mux.Unlock()
	c.mux.Lock()

	return c.limit
}

func (c *Count) add() error {
	defer c.mux.Unlock()
	c.mux.Lock()

	if c.limit > 0 && c.count >= c.limit {
		return ErrLimitReached
	}
	c.count++
	return nil
}

func (c *Count) sub() {
	defer c.mux.Unlock()
	c.mux.Lock()

	c.count--
}
