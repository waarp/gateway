package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// timingCost is high enough for a bcrypt comparison to take a measurable
// time (tens of milliseconds), and low enough to keep the tests fast.
const timingCost = 10

// minDuration returns the shortest duration of a few runs of fn, which
// filters out scheduling noise.
func minDuration(fn func()) time.Duration {
	best := time.Duration(1<<63 - 1)

	for range 3 {
		start := time.Now()

		fn()

		if d := time.Since(start); d < best {
			best = d
		}
	}

	return best
}

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("sesame"), timingCost)
	assert.NoError(t, err)

	t.Run("Given the right password", func(t *testing.T) {
		t.Parallel()
		assert.True(t, CheckPassword(timingCost, string(hash), "sesame"))
	})

	t.Run("Given a wrong password", func(t *testing.T) {
		t.Parallel()
		assert.False(t, CheckPassword(timingCost, string(hash), "not_sesame"))
	})

	t.Run("Given no reference hash", func(t *testing.T) {
		t.Parallel()
		assert.False(t, CheckPassword(timingCost, "", "sesame"))
		assert.False(t, CheckPassword(timingCost, "", ""))
	})

	t.Run("Given no reference hash, the check takes as long as with one", func(t *testing.T) {
		t.Parallel()

		CheckPassword(timingCost, "", "warm-up") // builds the dummy hash

		withHash := minDuration(func() { CheckPassword(timingCost, string(hash), "not_sesame") })
		withoutHash := minDuration(func() { CheckPassword(timingCost, "", "not_sesame") })

		assert.GreaterOrEqual(t, withoutHash, withHash/2,
			"no hash: %v, hash: %v", withoutHash, withHash)
	})
}
