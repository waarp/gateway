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
