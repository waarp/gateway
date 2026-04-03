package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_4AddEbicsContractRefreshPolicies(t *testing.T, eng *testEngine) Change {
	mig := Migrations[68]

	t.Run("When applying the 0.16.4 EBICS contract refresh policy migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_contract_refresh_policies"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS contract refresh policy table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_contract_refresh_policies"))
		})

		t.Run("Then it should enforce uniqueness on owner/name", func(t *testing.T) {
			eng.NoError(t, `INSERT INTO clients
				(id, owner, name, protocol)
				VALUES (10, 'gw', 'ebics-client-1', 'ebics'),
				       (11, 'gw', 'ebics-client-2', 'ebics')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM clients WHERE id IN (10, 11)`) })

			eng.NoError(t, `INSERT INTO ebics_hosts
				(id, owner, name, host_id, enabled, is_server, protocol_version, transport)
				VALUES (100, 'gw', 'host-1', 'HOST1', 1, 0, 'H005', 'https'),
				       (101, 'gw', 'host-2', 'HOST2', 1, 0, 'H005', 'https')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_hosts WHERE id IN (100, 101)`) })

			eng.NoError(t, `INSERT INTO ebics_subscribers
				(id, owner, ebics_host_id, name, partner_id, user_id, enabled)
				VALUES (20, 'gw', 100, 'partner-1:user-1', 'PARTNER1', 'USER1', 1),
				       (21, 'gw', 101, 'partner-2:user-2', 'PARTNER2', 'USER2', 1)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_subscribers WHERE id IN (20, 21)`) })

			eng.NoError(t, `INSERT INTO ebics_contract_refresh_policies
				(id, owner, name, enabled, client_id, ebics_subscriber_id, include_hev, interval_seconds, status, next_run_at)
				VALUES (1, 'gw', 'daily-bank-a', 1, 10, 20, 1, 86400, 'READY', CURRENT_TIMESTAMP)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_contract_refresh_policies`) })

			err := eng.Exec(`INSERT INTO ebics_contract_refresh_policies
				(id, owner, name, enabled, client_id, ebics_subscriber_id, include_hev, interval_seconds, status, next_run_at)
				VALUES (2, 'gw', 'daily-bank-a', 1, 11, 21, 1, 86400, 'READY', CURRENT_TIMESTAMP)`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_contract_refresh_policies"))
		})
	})

	return mig
}
