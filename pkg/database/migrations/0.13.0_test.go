package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testVer0_13_0AddTransferAutoResume(t *testing.T, eng *testEngine) Change {
	mig := Migrations[59]

	t.Run("When applying the 0.13.0 auto-resume columns addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "transfers",
			"remaining_tries", "next_retry_delay", "retry_increment_factor", "next_retry")

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers",
				"remaining_tries", "next_retry_delay", "retry_increment_factor", "next_retry")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")

			t.Run("Then it should have dropped the new columns", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfers",
					"remaining_tries", "next_retry_delay", "retry_increment_factor", "next_retry")
			})

			// Sanity check on the normalized_transfers view
			require.NoError(t, eng.DB.QueryRow(`SELECT * FROM normalized_transfers`).Err())
		})
	})

	return mig
}
