package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_0AddFilewatchers(t *testing.T, eng *testEngine) Change {
	mig := Migrations[64]

	t.Run("When applying the 0.16.0 filewatcher addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "file_watchers"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "file_watchers"))
			tableShouldHaveColumns(t, eng.DB, "file_watchers",
				"id", "owner", "disabled", "interval", "flow", "pattern",
				"remote_account_id", "client_id", "rule_id", "no_duplicate_check")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "file_watchers"))
			})
		})
	})

	return mig
}

func testVer0_16_0AddNormalizedInfo(t *testing.T, eng *testEngine) Change {
	mig := Migrations[65]

	t.Run("When applying the 0.16.0 normalized transfer info addition", func(t *testing.T) {
		require.False(t, doesViewExist(t, eng.DB, eng.Dialect, "normalized_transfer_info"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new view", func(t *testing.T) {
			assert.True(t, doesViewExist(t, eng.DB, eng.Dialect, "normalized_transfer_info"))
			tableShouldHaveColumns(t, eng.DB, "normalized_transfer_info",
				"owner_id", "name", "value")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the view", func(t *testing.T) {
				assert.False(t, doesViewExist(t, eng.DB, eng.Dialect, "normalized_transfer_info"))
			})
		})
	})

	return mig
}
