package pipeline

import (
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Signals is a map regrouping the signal channels of all ongoing transfers.
// The signal channel of a specific transfer can be retrieved from this map
// using the transfer's ID.
//nolint:gochecknoglobals // FIXME: could be refactored
var Signals = &SignalMap{mp: map[uint64]chan model.Signal{}}

// SignalMap is the type of the Signals map. It consists of a map associating
// transfer IDs to their signal channel. The structure contains a mutex so that
// the Add, SendSignal & Delete methods are thread-safe.
type SignalMap struct {
	mux sync.RWMutex
	mp  map[uint64]chan model.Signal
}

// Add creates a new signal channel and adds it to the internal map, along with
// the given ID. The new channel is then returned. If the given ID is already
// present in the map, the old channel is returned instead.
func (s *SignalMap) Add(id uint64) chan model.Signal {
	s.mux.Lock()
	defer s.mux.Unlock()

	if ch, ok := s.mp[id]; ok {
		return ch
	}

	ch := make(chan model.Signal)
	s.mp[id] = ch

	return ch
}

// SendSignal sends the given signal on the channel associated with the given ID.
// If the ID does not exist, the method does nothing.
func (s *SignalMap) SendSignal(id uint64, signal model.Signal) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	ch, ok := s.mp[id]
	if !ok {
		return
	}

	ch <- signal
}

// Delete closes the channel associated with the given ID, and then removes it
// from the map. If the ID does not exist, the method does nothing.
func (s *SignalMap) Delete(id uint64) {
	s.mux.Lock()
	defer s.mux.Unlock()

	ch, ok := s.mp[id]
	if !ok {
		return
	}

	close(ch)
	delete(s.mp, id)
}
