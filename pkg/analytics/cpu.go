package analytics

import (
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// cpuMeter measures the share of one CPU core the process takes between two
// samples: the CPU time it consumed over the time elapsed.
type cpuMeter struct {
	mu       sync.Mutex
	proc     *process.Process
	failed   bool // the process cannot be read: the share stays unmeasured
	lastAt   time.Time
	lastBusy float64 // CPU seconds consumed at the last sample
	share    float64
	measured bool
}

// sample reads the CPU time of the process at now and, from the second
// sample, the share of one core taken since the previous one.
func (m *cpuMeter) sample(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failed {
		return
	}

	if m.proc == nil {
		proc, err := process.NewProcess(int32(os.Getpid())) //nolint:gosec //a PID fits
		if err != nil {
			m.failed = true

			return
		}

		m.proc = proc
	}

	times, err := m.proc.Times()
	if err != nil {
		return // a passing failure: the next sample may read
	}

	busy := times.User + times.System

	if !m.lastAt.IsZero() {
		if elapsed := now.Sub(m.lastAt).Seconds(); elapsed > 0 {
			m.share, m.measured = (busy-m.lastBusy)/elapsed, true
		}
	}

	m.lastAt, m.lastBusy = now, busy
}

// value gives the share of one core (1 is a whole core) and whether it has
// been measured.
func (m *cpuMeter) value() (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.share, m.measured
}
