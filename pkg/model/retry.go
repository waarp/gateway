package model

import "math"

// minRetryDelay is the wait the gateway applies when retries are asked
// without a delay: one second, never an immediate retry.
const minRetryDelay int32 = 1

// FirstRetryDelay is the wait before the first retry of a transfer: the
// delay of the client, or one second when retries are asked without one.
func FirstRetryDelay(attempts int8, delay int32) int32 {
	if attempts != 0 && delay == 0 {
		return minRetryDelay
	}

	return delay
}

// NextRetryDelay is the wait before the retry that follows one that waited
// delay: the delay multiplied by the factor, truncated to the second and
// capped at the largest 32-bit value (the pipeline applies it after each
// failed attempt).
func NextRetryDelay(delay int32, factor float32) int32 {
	next := float32(delay) * factor
	if next > float32(math.MaxInt32) {
		return math.MaxInt32
	}

	return int32(next)
}

// RetryWaits are the waits before each of the automatic retries of a
// client, as the pipeline schedules them: the first delay, then each wait
// multiplied by the factor (1 when none is set). The web interface draws
// them; the pipeline and this function share FirstRetryDelay and
// NextRetryDelay, the single source of the computation.
func RetryWaits(attempts int8, delay int32, factor float32) []int32 {
	if attempts <= 0 {
		return nil
	}

	if factor == 0 {
		factor = 1
	}

	waits := make([]int32, 0, attempts)
	cur := FirstRetryDelay(attempts, delay)

	for range attempts {
		waits = append(waits, cur)
		cur = NextRetryDelay(cur, factor)
	}

	return waits
}
