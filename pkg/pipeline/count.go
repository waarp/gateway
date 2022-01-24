package pipeline

import (
	"math"
	"math/bits"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// errLimitReached is the error returned when a counter cannot be incremented
// because the maximum limit has been reached.
var errLimitReached = types.NewTransferError(types.TeExceededLimit, "transfer limit reached")

var (
	// TransferInCount counts the current and maximum number of concurrent incoming
	// transfers. A limit of 0 means no limit.
	//nolint:gochecknoglobals // FIXME: could be refactored
	TransferInCount = &count{}

	// TransferOutCount counts the current and maximum number of concurrent outgoing
	// transfers. A limit of 0 means no limit.
	//nolint:gochecknoglobals // FIXME: could be refactored
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

func (c *count) GetAvailable() (int, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.limit == 0 {
		return 0, false
	}

	available := c.limit - c.count

	if bits.UintSize == 64 { //nolint:gomnd // a constant would be unnecessary
		return int(available), true
	}

	if available <= math.MaxInt32 {
		return int(available), true
	}

	return math.MaxInt32, true
}
