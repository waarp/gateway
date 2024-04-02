package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testVer0_6_0AddTransferInfoIsHistory(t *testing.T, eng *testEngine) Change {
	mig := Migrations[16]

	t.Run("When applying the 0.6.0 transfer info 'is_history' addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "transfer_info", "is_history")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfer_info", "is_history")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfer_info", "is_history")
			})
		})
	})

	return mig
}
