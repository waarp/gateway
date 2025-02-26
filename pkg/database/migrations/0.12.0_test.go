package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_12_0AddCryptoKeys(t *testing.T, eng *testEngine) Change {
	mig := Migrations[56]

	t.Run("When applying the 0.12.0 crypto keys addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))
			tableShouldHaveColumns(t, eng.DB, "crypto_keys",
				"id", "name", "type", "key")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))
			})
		})
	})

	return mig
}
