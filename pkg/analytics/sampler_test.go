package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSamplerRates(t *testing.T) {
	const (
		period   = 5 * time.Second
		received = 5000 // bytes per period
		sent     = 1000
	)

	start := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

	var s sampler

	in, out := s.rates()
	assert.Zero(t, in, "no rate without a sample")
	assert.Zero(t, out)

	s.record(sample{at: start})

	in, out = s.rates()
	assert.Zero(t, in, "no rate with one sample")
	assert.Zero(t, out)

	s.record(sample{at: start.Add(period), received: received, sent: sent})

	in, out = s.rates()
	assert.InDelta(t, received/period.Seconds(), in, 0.01, "the rate between two samples")
	assert.InDelta(t, sent/period.Seconds(), out, 0.01)

	// The window slides: after eight periods, the samples of the last window
	// stay, the oldest one at most a window old.
	for i := 2; i <= 8; i++ {
		s.record(sample{at: start.Add(time.Duration(i) * period), received: int64(i) * received, sent: int64(i) * sent})
	}

	assert.Len(t, s.samples, int(SampleWindow/period)+1)
	assert.Equal(t, start.Add(2*period), s.samples[0].at)

	in, out = s.rates()
	assert.InDelta(t, received/period.Seconds(), in, 0.01, "the rate over the window")
	assert.InDelta(t, sent/period.Seconds(), out, 0.01)
}

func TestThroughput(t *testing.T) {
	start := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

	var a analytics

	a.Sample(start)
	a.BytesReceived.Add(3000)
	a.BytesSent.Add(600)
	a.Sample(start.Add(3 * time.Second))

	in, out := a.Throughput()
	assert.InDelta(t, 1000, in, 0.01)
	assert.InDelta(t, 200, out, 0.01)
}
