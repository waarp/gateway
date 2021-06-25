package service

import (
	"context"
	"sync"
)

// TransferInterrupter is the interface stored in the RunningTransfers map. The
// functions exposed by this interface can be used to interrupt running transfers.
type TransferInterrupter interface {
	// Pause pauses the transfer. It can be resumed later on command.
	Pause(context.Context) error

	// Interrupt stops the transfer because of a service shutdown. Transfer will
	// be resumed automatically when the service restarts.
	Interrupt(context.Context) error

	// Cancel cancels the transfer. Transfer will be moved to history and thus
	// cannot be resumed.
	Cancel(context.Context) error
}

// TransferMap is a map[uint64]TransferInterrupter which is safe for concurrent
// use. It is used to provide a list of currently running transfers, along with
// the functions which can be used to interrupt them.
//
// The ClientTransfers map contains all the currently running client pipeline.
// For server pipelines, each server should maintain a TransferMap of its own.
type TransferMap struct {
	m      map[uint64]TransferInterrupter
	mut    sync.Mutex
	closed bool
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

// Pause pauses the transfer the given transfer ID if it exists in the map. If
// the ID cannot be found, the returned boolean will be false.
func (t *TransferMap) Pause(ctx context.Context, id uint64) (bool, error) {
	t.mut.Lock()
	defer t.mut.Unlock()
	ti, ok := t.m[id]
	if !ok {
		return false, nil
	}
	return true, ti.Pause(ctx)
}

// Interrupt interrupts the transfer the given transfer ID if it exists in the
// map. If the ID cannot be found, the returned boolean will be false. If the
// transfer cannot be cancelled, an error is returned.
func (t *TransferMap) Interrupt(ctx context.Context, id uint64) (bool, error) {
	t.mut.Lock()
	defer t.mut.Unlock()
	ti, ok := t.m[id]
	if !ok {
		return false, nil
	}
	return true, ti.Interrupt(ctx)
}

// Cancel cancels the transfer the given transfer ID if it exists in the map.
// If the ID cannot be found, the returned boolean will be false. If the
// transfer cannot be cancelled, an error is returned.
func (t *TransferMap) Cancel(ctx context.Context, id uint64) (bool, error) {
	t.mut.Lock()
	defer t.mut.Unlock()
	ti, ok := t.m[id]
	if !ok {
		return false, nil
	}
	return true, ti.Cancel(ctx)
}

// Exists returns whether the given ID exists in the map.
func (t *TransferMap) Exists(id uint64) bool {
	t.mut.Lock()
	defer t.mut.Unlock()
	_, ok := t.m[id]
	return ok
}

// Delete removed the given transfer ID and it's associated pipeline from the map.
func (t *TransferMap) Delete(id uint64) {
	t.mut.Lock()
	defer t.mut.Unlock()
	delete(t.m, id)
}

// InterruptAll interrupts all the transfers in the TransferMap.
func (t *TransferMap) InterruptAll(ctx context.Context) error {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.closed = true
	wg := sync.WaitGroup{}
	for id, ti := range t.m {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ti.Interrupt(ctx)
		}()
		delete(t.m, id)
	}
	wg.Wait()
	return ctx.Err()
}
