package pipeline

import "sync"

// ClientTransfers is a synchronized map containing the pipelines of all currently
// running client transfers. It can be used to interrupt transfers using the various
// functions exposed by the TransferInterrupter interface.
var ClientTransfers = NewTransferMap()

// TransferMap is a map[uint64]TransferInterrupter which is safe for concurrent
// use. It is used to provide a list of currently running transfers, along with
// the functions which can be used to interrupt them.
//
// The ClientTransfers map contains all the currently running client pipeline.
// For server pipelines, each server should maintain a TransferMap of its own.
type TransferMap struct {
	m   map[uint64]TransferInterrupter
	mut sync.Mutex
}

// NewTransferMap initializes and returns a new TransferMap instance.
func NewTransferMap() *TransferMap {
	return &TransferMap{m: make(map[uint64]TransferInterrupter)}
}

// Add adds the given pipeline (TransferInterrupter), along with its transfer ID
// to the map.
func (t *TransferMap) Add(id uint64, ti TransferInterrupter) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.m[id] = ti
}

// Get returns the TransferInterrupter associated with the given transfer ID if
// it exists in the map. If the ID cannot be found, the returned boolean will be
// false.
func (t *TransferMap) Get(id uint64) (TransferInterrupter, bool) {
	t.mut.Lock()
	defer t.mut.Unlock()
	ti, ok := t.m[id]
	return ti, ok
}

// Delete removed the given transfer ID and it's associated pipeline from the map.
func (t *TransferMap) Delete(id uint64) {
	t.mut.Lock()
	defer t.mut.Unlock()
	delete(t.m, id)
}

// InterruptAll interrupts all the transfers in the TransferMap.
func (t *TransferMap) InterruptAll() {
	t.mut.Lock()
	defer t.mut.Unlock()
	for id, ti := range t.m {
		ti.Interrupt()
		delete(t.m, id)
	}
}
