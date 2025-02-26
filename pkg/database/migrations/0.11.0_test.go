package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_11_0AddSNMPServerConf(t *testing.T, eng *testEngine) Change {
	mig := Migrations[55]

	t.Run("When applying the 0.11.0 SNMP server conf addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_server_conf"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_server_conf"))
			tableShouldHaveColumns(t, eng.DB, "snmp_server_conf",
				"id", "owner", "local_udp_address", "community",
				"v3_only", "v3_username",
				"v3_auth_protocol", "v3_auth_passphrase",
				"v3_priv_protocol", "v3_priv_passphrase")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "snmp_server_conf"))
			})
		})
	})

	return mig
}
