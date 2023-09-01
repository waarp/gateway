package utils

import "time"

// WaitChan takes a channel and waits for a signal from it. When a signal is
// received, the function returns true. Otherwise, if no signal has been
// received after the given timeout has elapsed, the function returns false.
func WaitChan[T any](c chan T, timeout time.Duration) bool {
	select {
	case <-c:
		return true
	case <-time.After(timeout):
		return false
	}
}
