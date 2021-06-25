package utils

import (
	"context"
	"sync"
)

// ErrorStorage is a storage for an error value. The error value can be stored
// using the Store function, and then retrieved using the Get function. Only the
// first error will be stored. Subsequent calls to Store will be inconsequential.
// The storage is safe for concurrent use.
type ErrorStorage struct {
	done chan error
	mux  sync.Mutex
	err  error
	once sync.Once
}

// NewErrorStorage returns a new ErrorStorage instance.
func NewErrorStorage() *ErrorStorage {
	return &ErrorStorage{done: make(chan error)}
}

// Store stores the given error in the storage. If an error is already stored,
// the function does nothing. Store also releases all goroutines waiting on the
// channel returned by Wait if there are any.
func (e *ErrorStorage) Store(err error) {
	if err == nil {
		return
	}
	e.mux.Lock()
	defer e.mux.Unlock()
	e.once.Do(func() {
		defer close(e.done)
		e.err = err
	})
}

// StoreCtx stores the given error in the storage. If an error is already stored,
// the function does nothing. Unlike Store, StoreCtx waits until a goroutine has
// retrieved the stored error via the Wait channel (to unsure the error has been
// taken into account), or until the given context is cancelled.
func (e *ErrorStorage) StoreCtx(ctx context.Context, err error) {
	if err == nil {
		return
	}
	e.mux.Lock()
	defer e.mux.Unlock()
	e.once.Do(func() {
		defer close(e.done)
		e.err = err
		select {
		case e.done <- err:
		case <-ctx.Done():
		}
	})
}

// Get returns the stored error. If no error is stored, returns nil.
func (e *ErrorStorage) Get() error {
	e.mux.Lock()
	defer e.mux.Unlock()
	return e.err
}

// Wait returns a channel which the caller can then use to wait for an error to
// be stored. When an error is stored, this channel is closed, releasing all the
// goroutines waiting on it. If an error is stored using the StoreCtx function,
// said error will first be sent on this channel before closing it.
func (e *ErrorStorage) Wait() <-chan error {
	if e.done == nil {
		e.done = make(chan error)
	}
	return e.done
}

// Close releases all the goroutines waiting on the Wait channel but does not
// store any error. This function should only be used when no further errors are
// expected.
func (e *ErrorStorage) Close() {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.once.Do(func() {
		e.err = nil
		close(e.done)
	})
}
