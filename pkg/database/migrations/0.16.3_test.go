package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_3AddEbicsRuntimePolicies(t *testing.T, eng *testEngine) Change {
	mig := Migrations[67]

	t.Run("When applying the 0.16.3 EBICS runtime policy migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_runtime_policies"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS runtime policy table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_runtime_policies"))
		})

		t.Run("Then it should enforce uniqueness on owner/name", func(t *testing.T) {
			eng.NoError(t, `INSERT INTO ebics_runtime_policies
				(id, owner, name, enabled, maintenance_interval_seconds, transaction_retention_seconds, rtn_event_retention_seconds)
				VALUES (1, 'gw', 'default', 1, 21600, 604800, 2592000)`)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM ebics_runtime_policies`) })

			err := eng.Exec(`INSERT INTO ebics_runtime_policies
				(id, owner, name, enabled, maintenance_interval_seconds, transaction_retention_seconds, rtn_event_retention_seconds)
				VALUES (2, 'gw', 'default', 1, 21600, 604800, 2592000)`)
			require.Error(t, err)
			shouldBeUniqueViolationError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_runtime_policies"))
		})
	})

	return mig
}
