package dashboard

import (
	"runtime"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

// Node is what the health strip reads about the node: the analytics
// service of the gateway (GatewayNode), a stub in the tests. A figure that
// cannot be read says so, the strip then shows it unmeasured.
type Node interface {
	Version() string
	StartTime() (time.Time, bool)
	MemoryUsed() uint64
	MemoryTotal() (uint64, bool)
	Cores() int
	Connections() (incoming, outgoing int64)
}

// Monitors counts the partner monitors of the node: how many, how many in
// error.
type Monitors func() (total, failing int, err error)

// GatewayNode reads the analytics service of the gateway, when it runs.
type GatewayNode struct{}

func (GatewayNode) Version() string { return version.Num }

func (GatewayNode) StartTime() (time.Time, bool) {
	if s := analytics.GlobalService; s != nil {
		if started := s.StartTime.Load(); started != nil {
			return *started, true
		}
	}

	return time.Time{}, false
}

func (GatewayNode) MemoryUsed() uint64 {
	if s := analytics.GlobalService; s != nil {
		return s.GetUsedMemory()
	}

	return 0
}

func (GatewayNode) MemoryTotal() (uint64, bool) { return analytics.TotalMemory() }

func (GatewayNode) Cores() int { return runtime.NumCPU() }

func (GatewayNode) Connections() (incoming, outgoing int64) {
	if s := analytics.GlobalService; s != nil {
		return s.OpenIncomingConnections.Load(), s.OpenOutgoingConnections.Load()
	}

	return 0, 0
}
