package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_10_0AddSNMPMonitors(t *testing.T, eng *testEngine) Change {
	mig := Migrations[52]

	t.Run("When applying the 0.10.0 SNMP monitors addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_monitors"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_monitors"))
			tableShouldHaveColumns(t, eng.DB, "snmp_monitors",
				"id", "name", "owner",
				"udp_address", "snmp_version", "community", "use_informs",
				"snmp_v3_security",
				"snmp_v3_context_name", "snmp_v3_context_engine_id",
				"snmp_v3_auth_engine_id", "snmp_v3_auth_username",
				"snmp_v3_auth_protocol", "snmp_v3_auth_passphrase",
				"snmp_v3_priv_protocol", "snmp_v3_priv_passphrase")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_monitors"))
			})
		})
	})

	return mig
}

func ver0_10_0AddLocalAccountIPAddr(t *testing.T, eng *testEngine) Change {
	mig := Migrations[53]

	t.Run("When applying the 0.10.0 local agent IP addresses addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "local_accounts", "ip_addresses")

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "local_accounts", "ip_addresses")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "local_accounts", "ip_addresses")
			})
		})
	})

	return mig
}
