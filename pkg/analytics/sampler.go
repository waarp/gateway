package analytics

import (
	"sync"
	"time"
)

const (
	// SamplePeriod is how often the service samples its counters.
	SamplePeriod = 5 * time.Second
	// SampleWindow is the span the rates are averaged over.
	SampleWindow = 30 * time.Second
)

// sample is a reading of the counters at a moment.
type sample struct {
	at             time.Time
	received, sent int64
}

// sampler keeps the samples of the window, the oldest first, and gives the
// mean rates over them.
type sampler struct {
	mu      sync.Mutex
	samples []sample
}

// record keeps a sample and drops the ones older than the window.
func (s *sampler) record(smp sample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.samples = append(s.samples, smp)

	for len(s.samples) > 1 && smp.at.Sub(s.samples[0].at) > SampleWindow {
		s.samples = s.samples[1:]
	}
}

// rates gives the mean bytes per second received and sent between the oldest
// and the newest sample; zero before two samples.
func (s *sampler) rates() (received, sent float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.samples) < 2 { //nolint:mnd //two samples make a rate
		return 0, 0
	}

	first, last := s.samples[0], s.samples[len(s.samples)-1]

	elapsed := last.at.Sub(first.at).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	return float64(last.received-first.received) / elapsed, float64(last.sent-first.sent) / elapsed
}
