package snmp

import (
	"net"
	"testing"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getLocalAddr(tb testing.TB) net.Addr {
	tb.Helper()

	//nolint:gosec //this is just for testing
	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err, "Failed to start listener")

	defer require.NoError(tb, list.Close(), "Failed to stop listener")

	return list.Addr()
}

func makeTrapListener(tb testing.TB, version gosnmp.SnmpVersion,
) (<-chan *gosnmp.SnmpPacket, net.Addr) {
	tb.Helper()
	addr := getLocalAddr(tb)
	list := gosnmp.NewTrapListener()
	list.Params = &gosnmp.GoSNMP{Version: version}

	receivedTraps := make(chan *gosnmp.SnmpPacket, 10)
	list.OnNewTrap = func(pak *gosnmp.SnmpPacket, _ *net.UDPAddr) {
		receivedTraps <- pak
	}

	go list.Listen(addr.String())
	tb.Cleanup(list.Close)

	return receivedTraps, addr
}

func assertReceivedVar(tb testing.TB, receivedTraps <-chan *gosnmp.SnmpPacket,
	expectedVars ...gosnmp.SnmpPDU,
) {
	tb.Helper()

	trap := <-receivedTraps
	for _, expectedVar := range expectedVars {
		assert.Contains(tb, trap.Variables, expectedVar)
	}
}
