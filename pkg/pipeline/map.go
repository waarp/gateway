package pipeline

import "sync"

// ClientTransfers is a synchronized map containing the pipelines of all currently
// running client transfers. It can be used to interrupt transfers using the various
// functions exposed by the TransferInterrupter interface.
var ClientTransfers = NewTransferMap()

type TransferMap struct {
	m   map[uint64]TransferInterrupter
	mut sync.Mutex
}

func NewTransferMap() *TransferMap {
	return &TransferMap{m: make(map[uint64]TransferInterrupter)}
}

func (t *TransferMap) Add(id uint64, ti TransferInterrupter) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.m[id] = ti
}

func (t *TransferMap) Get(id uint64) (TransferInterrupter, bool) {
	t.mut.Lock()
	defer t.mut.Unlock()
	ti, ok := t.m[id]
	return ti, ok
}

func (t *TransferMap) Delete(id uint64) {
	t.mut.Lock()
	defer t.mut.Unlock()
	delete(t.m, id)
}

func (t *TransferMap) Iterate(f func(TransferInterrupter)) {
	t.mut.Lock()
	defer t.mut.Unlock()
	for _, ti := range t.m {
		f(ti)
	}
}
