package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSNMPExport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbServer1 := &snmp.ServerConfig{
		LocalUDPAddress: "1.1.1.1:1",
		Community:       "com1",
		SNMPv3Only:      false,
	}
	dbServer2 := &snmp.ServerConfig{
		LocalUDPAddress: "2.2.2.2:2",
		Community:       "com2",
		SNMPv3Only:      false,
	}

	require.NoError(t, db.Insert(dbServer1).Run())
	require.NoError(t, dbtest.ChangeOwner("other-gw", db.Insert(dbServer2).Run))

	dbMonitor1 := &snmp.MonitorConfig{
		Name:       "mon1",
		Version:    "SNMPv2",
		UDPAddress: "9.9.9.9:9",
		Community:  "com9",
		UseInforms: true,
	}
	dbMonitor2 := &snmp.MonitorConfig{
		Name:       "mon2",
		Version:    "SNMPv2",
		UDPAddress: "8.8.8.8:8",
		Community:  "com8",
		UseInforms: true,
	}
	dbMonitor3 := &snmp.MonitorConfig{
		Name:       "mon3",
		Version:    "SNMPv2",
		UDPAddress: "7.7.7.7:7",
		Community:  "com7",
		UseInforms: true,
	}

	require.NoError(t, db.Insert(dbMonitor1).Run())
	require.NoError(t, db.Insert(dbMonitor2).Run())
	require.NoError(t, dbtest.ChangeOwner("other-gw", db.Insert(dbMonitor3).Run))

	res, err := exportSNMPConfig(logger, db)
	require.NoError(t, err)

	t.Run("Exported server conf", func(t *testing.T) {
		require.NotNil(t, res.Server)

		assert.Equal(t, res.Server.LocalUDPAddress, dbServer1.LocalUDPAddress)
		assert.Equal(t, res.Server.Community, dbServer1.Community)
		assert.Equal(t, res.Server.V3Only, dbServer1.SNMPv3Only)
		assert.Equal(t, res.Server.V3Username, dbServer1.SNMPv3Username)
		assert.Equal(t, res.Server.V3AuthProtocol, dbServer1.SNMPv3AuthProtocol)
		assert.Equal(t, res.Server.V3AuthPassphrase, dbServer1.SNMPv3AuthPassphrase.String())
		assert.Equal(t, res.Server.V3PrivacyProtocol, dbServer1.SNMPv3PrivProtocol)
		assert.Equal(t, res.Server.V3PrivacyPassphrase, dbServer1.SNMPv3PrivPassphrase.String())
	})

	t.Run("Exported monitor config", func(t *testing.T) {
		require.Len(t, res.Monitors, 2)

		assert.Equal(t, res.Monitors[0].Name, dbMonitor1.Name)
		assert.Equal(t, res.Monitors[0].SNMPVersion, dbMonitor1.Version)
		assert.Equal(t, res.Monitors[0].UDPAddress, dbMonitor1.UDPAddress)
		assert.Equal(t, res.Monitors[0].Community, dbMonitor1.Community)
		assert.Equal(t, res.Monitors[0].UseInforms, dbMonitor1.UseInforms)
		assert.Equal(t, res.Monitors[0].V3ContextName, dbMonitor1.ContextName)
		assert.Equal(t, res.Monitors[0].V3ContextEngineID, dbMonitor1.ContextEngineID)
		assert.Equal(t, res.Monitors[0].V3Security, dbMonitor1.SNMPv3Security)
		assert.Equal(t, res.Monitors[0].V3AuthEngineID, dbMonitor1.AuthEngineID)
		assert.Equal(t, res.Monitors[0].V3AuthUsername, dbMonitor1.AuthUsername)
		assert.Equal(t, res.Monitors[0].V3AuthProtocol, dbMonitor1.AuthProtocol)
		assert.Equal(t, res.Monitors[0].V3AuthPassphrase, dbMonitor1.AuthPassphrase.String())
		assert.Equal(t, res.Monitors[0].V3PrivacyProtocol, dbMonitor1.PrivProtocol)
		assert.Equal(t, res.Monitors[0].V3PrivacyPassphrase, dbMonitor1.PrivPassphrase.String())

		assert.Equal(t, res.Monitors[1].Name, dbMonitor2.Name)
		assert.Equal(t, res.Monitors[1].SNMPVersion, dbMonitor2.Version)
		assert.Equal(t, res.Monitors[1].UDPAddress, dbMonitor2.UDPAddress)
		assert.Equal(t, res.Monitors[1].Community, dbMonitor2.Community)
		assert.Equal(t, res.Monitors[1].UseInforms, dbMonitor2.UseInforms)
		assert.Equal(t, res.Monitors[1].V3ContextName, dbMonitor2.ContextName)
		assert.Equal(t, res.Monitors[1].V3ContextEngineID, dbMonitor2.ContextEngineID)
		assert.Equal(t, res.Monitors[1].V3Security, dbMonitor2.SNMPv3Security)
		assert.Equal(t, res.Monitors[1].V3AuthEngineID, dbMonitor2.AuthEngineID)
		assert.Equal(t, res.Monitors[1].V3AuthUsername, dbMonitor2.AuthUsername)
		assert.Equal(t, res.Monitors[1].V3AuthProtocol, dbMonitor2.AuthProtocol)
		assert.Equal(t, res.Monitors[1].V3AuthPassphrase, dbMonitor2.AuthPassphrase.String())
		assert.Equal(t, res.Monitors[1].V3PrivacyProtocol, dbMonitor2.PrivProtocol)
		assert.Equal(t, res.Monitors[1].V3PrivacyPassphrase, dbMonitor2.PrivPassphrase.String())
	})
}
