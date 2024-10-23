package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_12_0AddPGPKeys(t *testing.T, eng *testEngine) Change {
	mig := Migrations[56]

	t.Run("When applying the 0.12.0 PGP keys addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "pgp_keys"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "pgp_keys"))
			tableShouldHaveColumns(t, eng.DB, "pgp_keys",
				"id", "name", "private_key", "public_key")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "pgp_keys"))
			})
		})
	})

	return mig
}
