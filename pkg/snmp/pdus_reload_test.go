package snmp

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

// Each sub-agent gets its own instances of the objects: building the server
// twice, as a reload of its configuration does, leaves the objects intact.
func TestServerPDUsKeepTheirOIDs(t *testing.T) {
	first := getServerPDUs()
	second := getServerPDUs()

	for i, pdu := range ServerPDUs {
		assert.Equal(t, pdu.OID+".0", first[i].OID, "first server")
		assert.Equal(t, pdu.OID+".0", second[i].OID, "second server")
	}
}

func getObject(tb testing.TB, addr, oid string) *gosnmp.SnmpPacket {
	tb.Helper()

	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(tb, err)
	port, err := strconv.ParseUint(portStr, 10, 16)
	require.NoError(tb, err)

	client := &gosnmp.GoSNMP{
		Target: host, Port: uint16(port), Community: "public",
		Version: gosnmp.Version2c, Timeout: 2 * time.Second, Retries: 1,
	}
	require.NoError(tb, client.Connect())
	tb.Cleanup(func() { _ = client.Conn.Close() })

	res, err := client.Get([]string{oid})
	require.NoError(tb, err)
	require.Len(tb, res.Variables, 1)

	return res
}

// The objects are still served after a reload of the server configuration.
func TestObjectsServedAfterReload(t *testing.T) {
	analytics.GlobalService = &analytics.Service{}
	t.Cleanup(func() { analytics.GlobalService = nil })

	db := dbtest.TestDatabase(t)
	config := ServerConfig{LocalUDPAddress: "127.0.0.1:0", Community: "public"}
	require.NoError(t, db.Insert(&config).Run())

	s := &Service{DB: db}
	require.NoError(t, s.Start())
	t.Cleanup(func() { require.NoError(t, s.Stop(context.Background())) })

	before := getObject(t, s.GetServerAddr(), AppVersionOID+".0")
	assert.Equal(t, gosnmp.OctetString, before.Variables[0].Type, "before the reload")

	require.NoError(t, s.ReloadServerConf(t.Context()))

	after := getObject(t, s.GetServerAddr(), AppVersionOID+".0")
	assert.Equal(t, gosnmp.OctetString, after.Variables[0].Type, "after the reload: %v", after.Variables[0])
}
