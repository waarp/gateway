package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_0AddEbicsTables(t *testing.T, eng *testEngine) Change {
	mig := Migrations[64]

	t.Run("When applying the 0.16.0 EBICS backend tables migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_hosts"))
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_rtn_events"))
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_transactions"))
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_nonces"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS tables", func(t *testing.T) {
			for _, table := range []string{
				"ebics_hosts",
				"ebics_subscribers",
				"ebics_operations",
				"ebics_contract_views",
				"ebics_contract_view_items",
				"ebics_rtn_events",
				"ebics_transactions",
				"ebics_transaction_segments",
				"ebics_nonces",
				"ebics_key_lifecycles",
				"ebics_initialization_workflows",
				"ebics_rtn_providers",
			} {
				assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, table), "missing table %s", table)
			}
		})

		t.Run("Then it should enforce uniqueness and cascade semantics", func(t *testing.T) {
			eng.NoError(t, `INSERT INTO ebics_hosts
				(id, owner, name, host_id, enabled, is_server, protocol_version, transport)
				VALUES (1, 'gw', 'host-1', 'HOST1', 1, 1, 'H005', 'https')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_hosts`) })

			eng.NoError(t, `INSERT INTO ebics_subscribers
				(id, owner, ebics_host_id, name, partner_id, user_id, enabled)
				VALUES (10, 'gw', 1, 'partner-1:user-1', 'PARTNER1', 'USER1', 1)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_subscribers`) })

			eng.NoError(t, `INSERT INTO ebics_rtn_events
				(id, owner, source, idempotence_key, status, received_at, ebics_host_id, ebics_subscriber_id)
				VALUES (20, 'gw', 'wss', 'idem-1', 'RECEIVED', '2026-04-01 10:00:00', 1, 10)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_rtn_events`) })

			err := eng.Exec(`INSERT INTO ebics_rtn_events
				(id, owner, source, idempotence_key, status, received_at, ebics_host_id, ebics_subscriber_id)
				VALUES (21, 'gw', 'wss', 'idem-1', 'DUPLICATE', '2026-04-01 10:01:00', 1, 10)`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)

			eng.NoError(t, `INSERT INTO ebics_transactions
				(id, owner, ebics_host_id, ebics_subscriber_id, transaction_id, order_type, status, direction)
				VALUES (30, 'gw', 1, 10, 'TX-1', 'BTU', 'RUNNING', 'INBOUND')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_transactions`) })

			err = eng.Exec(`INSERT INTO ebics_transactions
				(id, owner, ebics_host_id, ebics_subscriber_id, transaction_id, order_type, status, direction)
				VALUES (31, 'gw', 1, 10, 'TX-1', 'BTU', 'RUNNING', 'INBOUND')`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)

			eng.NoError(t, `INSERT INTO ebics_transaction_segments
				(id, owner, ebics_transaction_id, segment_number, segment_status)
				VALUES (40, 'gw', 30, 1, 'STORED')`)

			err = eng.Exec(`INSERT INTO ebics_transaction_segments
				(id, owner, ebics_transaction_id, segment_number, segment_status)
				VALUES (41, 'gw', 30, 1, 'STORED')`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)

			eng.NoError(t, `INSERT INTO ebics_nonces
				(id, owner, ebics_subscriber_id, nonce, timestamp, expires_at)
				VALUES (50, 'gw', 10, 'nonce-1', '2026-04-01 10:00:00', '2026-04-01 10:15:00')`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_nonces`) })

			err = eng.Exec(`INSERT INTO ebics_nonces
				(id, owner, ebics_subscriber_id, nonce, timestamp, expires_at)
				VALUES (51, 'gw', 10, 'nonce-1', '2026-04-01 10:01:00', '2026-04-01 10:16:00')`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)

			eng.NoError(t, `DELETE FROM ebics_transactions WHERE id=30`)

			var count int
			require.NoError(t, eng.DB.QueryRow(
				`SELECT COUNT(*) FROM ebics_transaction_segments WHERE ebics_transaction_id=30`,
			).Scan(&count))
			require.Zero(t, count)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")

			t.Run("Then it should drop the EBICS tables", func(t *testing.T) {
				for _, table := range []string{
					"ebics_hosts",
					"ebics_subscribers",
					"ebics_operations",
					"ebics_contract_views",
					"ebics_contract_view_items",
					"ebics_rtn_events",
					"ebics_transactions",
					"ebics_transaction_segments",
					"ebics_nonces",
					"ebics_key_lifecycles",
					"ebics_initialization_workflows",
					"ebics_rtn_providers",
				} {
					assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, table), "table %s should be dropped", table)
				}
			})
		})
	})

	return mig
}
