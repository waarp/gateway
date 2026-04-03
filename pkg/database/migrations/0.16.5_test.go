package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_5AddEbicsHistoryEntries(t *testing.T, eng *testEngine) Change {
	mig := Migrations[69]

	t.Run("When applying the 0.16.5 EBICS history migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_history_entries"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS history table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_history_entries"))
		})

		t.Run("Then it should enforce host/subscriber consistency through foreign keys", func(t *testing.T) {
			eng.NoError(t, `INSERT INTO clients (id, owner, name, protocol) VALUES (31, 'gw', 'ebics-client-history', 'ebics')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM clients WHERE id=31`) })

			eng.NoError(t, `INSERT INTO ebics_hosts
				(id, owner, name, host_id, enabled, is_server, protocol_version, transport)
				VALUES (301, 'gw', 'host-history', 'HOST-HISTORY', 1, 0, 'H005', 'https')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_hosts WHERE id=301`) })

			eng.NoError(t, `INSERT INTO ebics_subscribers
				(id, owner, ebics_host_id, name, partner_id, user_id, enabled)
				VALUES (401, 'gw', 301, 'partner:user', 'PARTNER', 'USER', 1)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_subscribers WHERE id=401`) })

			eng.NoError(t, `INSERT INTO ebics_history_entries
				(id, owner, history_type, operation_type, status, client_id, ebics_host_id, ebics_subscriber_id, evidence, metadata)
				VALUES (1, 'gw', 'ACTION', 'INITIALIZATION', 'CANCELLED', 31, 301, 401, '{}', '{}')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_history_entries`) })

			err := eng.Exec(`INSERT INTO ebics_history_entries
				(id, owner, history_type, operation_type, status, client_id, ebics_host_id, ebics_subscriber_id, evidence, metadata)
				VALUES (2, 'gw', 'ACTION', 'INITIALIZATION', 'CANCELLED', 31, 9999, 401, '{}', '{}')`)
			require.Error(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_history_entries"))
		})
	})

	return mig
}
