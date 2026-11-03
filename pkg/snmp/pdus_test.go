package snmp

import (
	"testing"
	"time"

	snmplib "github.com/slayercat/GoSNMPServer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
)

func serverPDU(tb testing.TB, oid string) *snmplib.PDUValueControlItem {
	tb.Helper()

	for _, pdu := range ServerPDUs {
		if pdu.OID == oid {
			return pdu
		}
	}

	tb.Fatalf("no server PDU with the OID %q", oid)

	return nil
}

// The throughput objects give the mean rates of the sampler, in kB/s (#828).
func TestThroughputPDUs(t *testing.T) {
	analytics.GlobalService = &analytics.Service{}
	t.Cleanup(func() { analytics.GlobalService = nil })

	start := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	analytics.GlobalService.Sample(start)
	analytics.AddBytesReceived(50_000)
	analytics.AddBytesSent(20_000)
	analytics.GlobalService.Sample(start.Add(10 * time.Second))

	received, err := serverPDU(t, ResObjectIncomingThroughputOID).OnGet()
	require.NoError(t, err)
	assert.Equal(t, uint32(5), received)

	sent, err := serverPDU(t, ResObjectOutgoingThroughputOID).OnGet()
	require.NoError(t, err)
	assert.Equal(t, uint32(2), sent)
}

func TestThroughputPDUsWithoutAnalytics(t *testing.T) {
	analytics.GlobalService = nil

	_, err := serverPDU(t, ResObjectIncomingThroughputOID).OnGet()
	require.ErrorIs(t, err, ErrNoAnalytics)
}
