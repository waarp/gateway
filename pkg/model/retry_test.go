package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstRetryDelay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(0), FirstRetryDelay(0, 0), "no retry, no delay")
	assert.Equal(t, int32(1), FirstRetryDelay(3, 0), "retries without a delay wait one second")
	assert.Equal(t, int32(30), FirstRetryDelay(3, 30))
}

func TestNextRetryDelay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(60), NextRetryDelay(30, 2))
	assert.Equal(t, int32(30), NextRetryDelay(30, 1))
	assert.Equal(t, int32(45), NextRetryDelay(30, 1.5))
	assert.Equal(t, int32(7), NextRetryDelay(5, 1.5), "truncated to the second")
	assert.Equal(t, int32(0), NextRetryDelay(1, 0.5), "a factor below one may reach zero")
	assert.Equal(t, int32(math.MaxInt32), NextRetryDelay(math.MaxInt32, 2), "capped")
	assert.Equal(t, int32(math.MaxInt32), NextRetryDelay(1<<30, 4), "capped at the first overflow")
}

func TestRetryWaits(t *testing.T) {
	t.Parallel()

	assert.Nil(t, RetryWaits(0, 30, 2), "no retry")
	assert.Equal(t, []int32{30}, RetryWaits(1, 30, 2))
	assert.Equal(t, []int32{30, 60, 120, 240, 480}, RetryWaits(5, 30, 2))
	assert.Equal(t, []int32{30, 30, 30}, RetryWaits(3, 30, 0), "no factor stored: constant delay")
	assert.Equal(t, []int32{30, 45, 67, 100}, RetryWaits(4, 30, 1.5), "truncated at each step")
	assert.Equal(t, []int32{10, 5, 2, 1, 0}, RetryWaits(5, 10, 0.5), "a factor below one shrinks the waits")
	assert.Equal(t, []int32{1, 1, 1}, RetryWaits(3, 0, 1), "an empty delay is one second")

	waits := RetryWaits(127, 30, 2)
	assert.Len(t, waits, 127)
	assert.Equal(t, int32(math.MaxInt32), waits[126], "the waits are capped")
}
