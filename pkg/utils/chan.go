package utils

import (
	"context"
	"errors"
	"io"
	"time"
)

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

// GoRun is a convenience function to run a function in a goroutine and return
// a channel to receive the result.
func GoRun[T any](f func() T) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		ch <- f()
	}()

	return ch
}

func RunWithCtx(ctx context.Context, f func() error) error {
	done := GoRun(f)

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return context.Cause(ctx) //nolint:wrapcheck //wrapping adds nothing here
	}
}

func CheckCtx(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return context.Cause(ctx) //nolint:wrapcheck //wrapping adds nothing here
	default:
		return nil
	}
}

func RWWithCtx(ctx context.Context, f func([]byte) (int, error), b []byte) (int, error) {
	if err := CheckCtx(ctx); err != nil {
		return 0, err
	}

	n, rwErr := f(b)
	if rwErr != nil && !errors.Is(rwErr, io.EOF) {
		return n, rwErr
	}

	if err := CheckCtx(ctx); err != nil {
		return 0, err
	}

	return n, rwErr
}

func RWatWithCtx(ctx context.Context, f func([]byte, int64) (int, error), b []byte, off int64) (int, error) {
	if err := CheckCtx(ctx); err != nil {
		return 0, err
	}

	n, rwErr := f(b, off)
	if rwErr != nil && !errors.Is(rwErr, io.EOF) {
		return n, rwErr
	}

	if err := CheckCtx(ctx); err != nil {
		return 0, err
	}

	return n, rwErr
}

func TrySend[T any](c chan<- T, v T) bool {
	select {
	case c <- v:
		return true
	default:
		return false
	}
}

func TryRecv[T any](c <-chan T) (T, bool) {
	select {
	case v := <-c:
		return v, true
	default:
		return *new(T), false
	}
}
