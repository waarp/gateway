package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testVer0_14_0AddTransferStop(t *testing.T, eng *testEngine) Change {
	mig := Migrations[63]

	t.Run("When applying the 0.14.0 transfers stop addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "transfers", "stop")

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "stop")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")

			t.Run("Then it should have dropped the new columns", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfers", "stop")
			})

			// Sanity check on the normalized_transfers view
			row := eng.DB.QueryRow(`SELECT * FROM normalized_transfers`)
			defer row.Scan([]any{}...)
			require.NoError(t, row.Err())
		})
	})

	return mig
}
