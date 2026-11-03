package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// busy takes the processor for the duration.
func busy(d time.Duration) {
	for end := time.Now().Add(d); time.Now().Before(end); { //nolint:revive //an empty loop is the point
	}
}

func TestCPUMeter(t *testing.T) {
	var m cpuMeter

	_, measured := m.value()
	assert.False(t, measured, "nothing measured before two samples")

	m.sample(time.Now())

	_, measured = m.value()
	assert.False(t, measured, "nothing measured after one sample")

	busy(100 * time.Millisecond)
	m.sample(time.Now())

	share, measured := m.value()
	assert.True(t, measured)
	assert.Positive(t, share, "a busy loop takes some of a core")
}
