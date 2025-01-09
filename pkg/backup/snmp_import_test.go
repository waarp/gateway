package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestNewSNMPConfImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbServer2 := &snmp.ServerConfig{
		LocalUDPAddress: "2.2.2.2:2",
		Community:       "com2",
		SNMPv3Only:      false,
	}
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

	require.NoError(t, db.Insert(dbMonitor1).Run())
	require.NoError(t, dbtest.ChangeOwner("other-gw", db.Insert(dbMonitor2).Run))

	t.Run("No reset", func(t *testing.T) {
		const newMonitorName = "mon7"

		config := &file.SNMPConfig{
			Server: &file.SNMPServer{
				LocalUDPAddress:     "1.1.1.1:1",
				Community:           "com1",
				V3Only:              true,
				V3Username:          "toto",
				V3AuthProtocol:      "SHA",
				V3AuthPassphrase:    "foo",
				V3PrivacyProtocol:   "AES",
				V3PrivacyPassphrase: "bar",
			},
			Monitors: []*file.SNMPMonitor{{
				Name:                newMonitorName,
				SNMPVersion:         "SNMPv3",
				UDPAddress:          "7.7.7.7:7",
				Community:           "com7",
				UseInforms:          false,
				V3ContextName:       "cont",
				V3ContextEngineID:   "cont-id",
				V3Security:          "authPriv",
				V3AuthEngineID:      "auth-id",
				V3AuthUsername:      "titi",
				V3AuthProtocol:      "MD5",
				V3AuthPassphrase:    "fizz",
				V3PrivacyProtocol:   "DES",
				V3PrivacyPassphrase: "buzz",
			}},
		}

		require.NoError(t, importSNMPConfig(logger, db, config, false))

		t.Run("New server is imported", func(t *testing.T) {
			var check snmp.ServerConfig
			require.NoError(t, db.Get(&check, "owner=?", conf.GlobalConfig.GatewayName).Run())
			assert.Equal(t, check.LocalUDPAddress, config.Server.LocalUDPAddress)
			assert.Equal(t, check.Community, config.Server.Community)
			assert.Equal(t, check.SNMPv3Only, config.Server.V3Only)
			assert.Equal(t, check.SNMPv3Username, config.Server.V3Username)
			assert.Equal(t, check.SNMPv3AuthProtocol, config.Server.V3AuthProtocol)
			assert.Equal(t, check.SNMPv3AuthPassphrase.String(), config.Server.V3AuthPassphrase)
			assert.Equal(t, check.SNMPv3PrivProtocol, config.Server.V3PrivacyProtocol)
			assert.Equal(t, check.SNMPv3PrivPassphrase.String(), config.Server.V3PrivacyPassphrase)
		})

		t.Run("New monitor is imported", func(t *testing.T) {
			var check snmp.MonitorConfig
			require.NoError(t, db.Get(&check, "name=?", newMonitorName).Run())
			assert.Equal(t, check.Name, config.Monitors[0].Name)
			assert.Equal(t, check.Version, config.Monitors[0].SNMPVersion)
			assert.Equal(t, check.UDPAddress, config.Monitors[0].UDPAddress)
			assert.Equal(t, check.Community, config.Monitors[0].Community)
			assert.Equal(t, check.UseInforms, config.Monitors[0].UseInforms)
			assert.Equal(t, check.ContextName, config.Monitors[0].V3ContextName)
			assert.Equal(t, check.ContextEngineID, config.Monitors[0].V3ContextEngineID)
			assert.Equal(t, check.SNMPv3Security, config.Monitors[0].V3Security)
			assert.Equal(t, check.AuthEngineID, config.Monitors[0].V3AuthEngineID)
			assert.Equal(t, check.AuthUsername, config.Monitors[0].V3AuthUsername)
			assert.Equal(t, check.AuthProtocol, config.Monitors[0].V3AuthProtocol)
			assert.Equal(t, check.AuthPassphrase.String(), config.Monitors[0].V3AuthPassphrase)
			assert.Equal(t, check.PrivProtocol, config.Monitors[0].V3PrivacyProtocol)
			assert.Equal(t, check.PrivPassphrase.String(), config.Monitors[0].V3PrivacyPassphrase)
		})

		t.Run("Existing server is untouched", func(t *testing.T) {
			var check model.Slice[*snmp.ServerConfig]
			require.NoError(t, db.Select(&check).Where("owner<>?",
				conf.GlobalConfig.GatewayName).OrderBy("id", true).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbServer2)
		})

		t.Run("Existing monitors are untouched", func(t *testing.T) {
			var check model.Slice[*snmp.MonitorConfig]
			require.NoError(t, db.Select(&check).Where("name<>?", newMonitorName).
				OrderBy("id", true).Run())
			require.Len(t, check, 2)
			assert.Equal(t, check[0], dbMonitor1)
			assert.Equal(t, check[1], dbMonitor2)
		})
	})

	t.Run("With reset", func(t *testing.T) {
		const newMonitorName = "mon6"

		config := &file.SNMPConfig{
			Server: &file.SNMPServer{
				LocalUDPAddress:     "1.1.1.1:1",
				Community:           "com1",
				V3Only:              true,
				V3Username:          "toto",
				V3AuthProtocol:      "SHA",
				V3AuthPassphrase:    "foo",
				V3PrivacyProtocol:   "AES",
				V3PrivacyPassphrase: "bar",
			},
			Monitors: []*file.SNMPMonitor{{
				Name:                newMonitorName,
				SNMPVersion:         "SNMPv3",
				UDPAddress:          "7.7.7.7:7",
				Community:           "com7",
				UseInforms:          false,
				V3ContextName:       "cont",
				V3ContextEngineID:   "cont-id",
				V3Security:          "authPriv",
				V3AuthEngineID:      "auth-id",
				V3AuthUsername:      "titi",
				V3AuthProtocol:      "MD5",
				V3AuthPassphrase:    "fizz",
				V3PrivacyProtocol:   "DES",
				V3PrivacyPassphrase: "buzz",
			}},
		}

		require.NoError(t, importSNMPConfig(logger, db, config, true))

		t.Run("New server is imported", func(t *testing.T) {
			var check snmp.ServerConfig
			require.NoError(t, db.Get(&check, "owner=?", conf.GlobalConfig.GatewayName).Run())
			assert.Equal(t, check.LocalUDPAddress, config.Server.LocalUDPAddress)
			assert.Equal(t, check.Community, config.Server.Community)
			assert.Equal(t, check.SNMPv3Only, config.Server.V3Only)
			assert.Equal(t, check.SNMPv3Username, config.Server.V3Username)
			assert.Equal(t, check.SNMPv3AuthProtocol, config.Server.V3AuthProtocol)
			assert.Equal(t, check.SNMPv3AuthPassphrase.String(), config.Server.V3AuthPassphrase)
			assert.Equal(t, check.SNMPv3PrivProtocol, config.Server.V3PrivacyProtocol)
			assert.Equal(t, check.SNMPv3PrivPassphrase.String(), config.Server.V3PrivacyPassphrase)
		})

		t.Run("New monitor is imported", func(t *testing.T) {
			var check snmp.MonitorConfig
			require.NoError(t, db.Get(&check, "name=?", newMonitorName).Run())
			assert.Equal(t, check.Name, config.Monitors[0].Name)
			assert.Equal(t, check.Version, config.Monitors[0].SNMPVersion)
			assert.Equal(t, check.UDPAddress, config.Monitors[0].UDPAddress)
			assert.Equal(t, check.Community, config.Monitors[0].Community)
			assert.Equal(t, check.UseInforms, config.Monitors[0].UseInforms)
			assert.Equal(t, check.ContextName, config.Monitors[0].V3ContextName)
			assert.Equal(t, check.ContextEngineID, config.Monitors[0].V3ContextEngineID)
			assert.Equal(t, check.SNMPv3Security, config.Monitors[0].V3Security)
			assert.Equal(t, check.AuthEngineID, config.Monitors[0].V3AuthEngineID)
			assert.Equal(t, check.AuthUsername, config.Monitors[0].V3AuthUsername)
			assert.Equal(t, check.AuthProtocol, config.Monitors[0].V3AuthProtocol)
			assert.Equal(t, check.AuthPassphrase.String(), config.Monitors[0].V3AuthPassphrase)
			assert.Equal(t, check.PrivProtocol, config.Monitors[0].V3PrivacyProtocol)
			assert.Equal(t, check.PrivPassphrase.String(), config.Monitors[0].V3PrivacyPassphrase)
		})

		t.Run("Existing monitors are gone", func(t *testing.T) {
			var check model.Slice[*snmp.MonitorConfig]
			require.NoError(t, db.Select(&check).Where("name<>?", newMonitorName).
				Where("owner=?", conf.GlobalConfig.GatewayName).OrderBy("id", true).Run())
			assert.Empty(t, check)
		})
	})
}

func TestExistingSNMPConfImport(t *testing.T) {
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

	require.NoError(t, db.Insert(dbMonitor1).Run())
	require.NoError(t, dbtest.ChangeOwner("other-gw", db.Insert(dbMonitor2).Run))

	t.Run("No reset", func(t *testing.T) {
		newMonitorName := dbMonitor1.Name

		config := &file.SNMPConfig{
			Server: &file.SNMPServer{
				LocalUDPAddress:     "1.1.1.1:11",
				Community:           "com11",
				V3Only:              true,
				V3Username:          "toto",
				V3AuthProtocol:      "SHA",
				V3AuthPassphrase:    "foo",
				V3PrivacyProtocol:   "AES",
				V3PrivacyPassphrase: "bar",
			},
			Monitors: []*file.SNMPMonitor{{
				Name:                newMonitorName,
				SNMPVersion:         "SNMPv3",
				UDPAddress:          "7.7.7.7:7",
				Community:           "com7",
				UseInforms:          false,
				V3ContextName:       "cont",
				V3ContextEngineID:   "cont-id",
				V3Security:          "authPriv",
				V3AuthEngineID:      "auth-id",
				V3AuthUsername:      "titi",
				V3AuthProtocol:      "MD5",
				V3AuthPassphrase:    "fizz",
				V3PrivacyProtocol:   "DES",
				V3PrivacyPassphrase: "buzz",
			}},
		}

		require.NoError(t, importSNMPConfig(logger, db, config, false))

		t.Run("Server is updated", func(t *testing.T) {
			var check snmp.ServerConfig
			require.NoError(t, db.Get(&check, "owner=?", conf.GlobalConfig.GatewayName).Run())
			assert.Equal(t, check.LocalUDPAddress, config.Server.LocalUDPAddress)
			assert.Equal(t, check.Community, config.Server.Community)
			assert.Equal(t, check.SNMPv3Only, config.Server.V3Only)
			assert.Equal(t, check.SNMPv3Username, config.Server.V3Username)
			assert.Equal(t, check.SNMPv3AuthProtocol, config.Server.V3AuthProtocol)
			assert.Equal(t, check.SNMPv3AuthPassphrase.String(), config.Server.V3AuthPassphrase)
			assert.Equal(t, check.SNMPv3PrivProtocol, config.Server.V3PrivacyProtocol)
			assert.Equal(t, check.SNMPv3PrivPassphrase.String(), config.Server.V3PrivacyPassphrase)
		})

		t.Run("New monitor is imported", func(t *testing.T) {
			var check snmp.MonitorConfig
			require.NoError(t, db.Get(&check, "name=?", newMonitorName).Run())
			assert.Equal(t, check.Name, config.Monitors[0].Name)
			assert.Equal(t, check.Version, config.Monitors[0].SNMPVersion)
			assert.Equal(t, check.UDPAddress, config.Monitors[0].UDPAddress)
			assert.Equal(t, check.Community, config.Monitors[0].Community)
			assert.Equal(t, check.UseInforms, config.Monitors[0].UseInforms)
			assert.Equal(t, check.ContextName, config.Monitors[0].V3ContextName)
			assert.Equal(t, check.ContextEngineID, config.Monitors[0].V3ContextEngineID)
			assert.Equal(t, check.SNMPv3Security, config.Monitors[0].V3Security)
			assert.Equal(t, check.AuthEngineID, config.Monitors[0].V3AuthEngineID)
			assert.Equal(t, check.AuthUsername, config.Monitors[0].V3AuthUsername)
			assert.Equal(t, check.AuthProtocol, config.Monitors[0].V3AuthProtocol)
			assert.Equal(t, check.AuthPassphrase.String(), config.Monitors[0].V3AuthPassphrase)
			assert.Equal(t, check.PrivProtocol, config.Monitors[0].V3PrivacyProtocol)
			assert.Equal(t, check.PrivPassphrase.String(), config.Monitors[0].V3PrivacyPassphrase)
		})

		t.Run("Other server is untouched", func(t *testing.T) {
			var check model.Slice[*snmp.ServerConfig]
			require.NoError(t, db.Select(&check).Where("owner<>?",
				conf.GlobalConfig.GatewayName).OrderBy("id", true).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbServer2)
		})

		t.Run("Other monitors are untouched", func(t *testing.T) {
			var check model.Slice[*snmp.MonitorConfig]
			require.NoError(t, db.Select(&check).Where("name<>?", newMonitorName).
				OrderBy("id", true).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbMonitor2)
		})
	})
}
