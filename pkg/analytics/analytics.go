// Package analytics contains the code for the analytics service which collects
// statistics about the application, which can then be used by external interfaces.
package analytics

import (
	"runtime"
	"sync/atomic"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type analytics struct {
	StartTime atomic.Pointer[time.Time] // App launch time.

	RunningTransfers atomic.Int64 // Number of running transfers.

	OpenIncomingConnections atomic.Int64 // Number of open incoming connections.
	OpenOutgoingConnections atomic.Int64 // Number of open outgoing connections.

	// The bytes of the transfers written to the files of the gateway (received)
	// and read from them (sent) since its launch; the sampler gives their rates.
	BytesReceived atomic.Int64
	BytesSent     atomic.Int64

	sampler sampler
	cpu     cpuMeter
	dirs    dirs
}

// Sample records the counters at now, for the rates; the service does it every
// SamplePeriod.
func (a *analytics) Sample(now time.Time) {
	a.sampler.record(sample{at: now, received: a.BytesReceived.Load(), sent: a.BytesSent.Load()})
	a.cpu.sample(now)
}

// CPUShare is the share of one CPU core the gateway took between the last
// two samples (1 is a whole core), and whether it has been measured.
func (a *analytics) CPUShare() (share float64, measured bool) {
	return a.cpu.value()
}

// Throughput is the mean bytes per second received and sent over the last
// SampleWindow; zero before two samples.
func (a *analytics) Throughput() (received, sent float64) {
	return a.sampler.rates()
}

func (*analytics) GetVersion() string { return version.Num }

func (*analytics) GetUsedMemory() uint64 {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	return stats.Sys
}

func AddIncomingConnection() {
	if GlobalService != nil {
		GlobalService.OpenIncomingConnections.Add(1)
	}
}

func SubIncomingConnection() {
	if GlobalService != nil {
		GlobalService.OpenIncomingConnections.Add(-1)
	}
}

func AddOutgoingConnection() {
	if GlobalService != nil {
		GlobalService.OpenOutgoingConnections.Add(1)
	}
}

func SubOutgoingConnection() {
	if GlobalService != nil {
		GlobalService.OpenOutgoingConnections.Add(-1)
	}
}

// AddBytesReceived counts n bytes written to a file by a transfer.
func AddBytesReceived(n int) {
	if GlobalService != nil {
		GlobalService.BytesReceived.Add(int64(n))
	}
}

// AddBytesSent counts n bytes read from a file by a transfer.
func AddBytesSent(n int) {
	if GlobalService != nil {
		GlobalService.BytesSent.Add(int64(n))
	}
}
