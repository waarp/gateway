package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_4_2RemoveHistoryRemoteIDUnique(t *testing.T, eng *testEngine) Change {
	mig := Migrations[1]

	t.Run("When applying the 0.4.2 history unique constraint removal", func(t *testing.T) {
		assert.True(t,
			doesIndexExist(t, eng.DB, eng.Dialect, "transfer_history", "UQE_transfer_history_histRemID"),
			"Before the migration, the history unique constraint should exist")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have removed the history unique constraint", func(t *testing.T) {
			assert.False(t,
				doesIndexExist(t, eng.DB, eng.Dialect, "transfer_history",
					"UQE_transfer_history_histRemID"),
				"After the migration, the history unique constraint should no longer exist")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			assert.True(t,
				doesIndexExist(t, eng.DB, eng.Dialect, "transfer_history", "UQE_transfer_history_histRemID"),
				"After reverting the migration, the history unique constraint should exist again")
		})
	})

	return mig
}
