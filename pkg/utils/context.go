package utils

import "context"

// CanceledContext returns a context instance which has already expired (this
// can be seen as the opposite of context.Background() which is never expires).
// This can be useful in tests or other cases where a context is needed, but
// the outcome of an action does not matter.
func CanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}
