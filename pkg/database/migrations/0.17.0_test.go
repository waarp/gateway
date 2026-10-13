package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_17_0TransferInfoValueType(t *testing.T, eng *testEngine) Change {
	mig := Migrations[66]

	// Migration only affects MySQL, skip other RDBMS.
	if eng.Dialect != MySQL {
		return mig
	}

	t.Run("When applying the 0.17.0 transfer info value type change", func(t *testing.T) {
		require.Equal(t, "TEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
			`The "value" column should be of type TEXT before the migration`)

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have changed the column type", func(t *testing.T) {
			assert.Equal(t, "LONGTEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
				`The "value" column should now be of type LONGTEXT`)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the column type", func(t *testing.T) {
				assert.Equal(t, "TEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
					`The "value" column should be of type TEXT again`)
			})
		})
	})

	return mig
}
