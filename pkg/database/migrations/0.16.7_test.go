package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_7AddEbicsServerReportingProjectionTables(t *testing.T, eng *testEngine) Change {
	mig := Migrations[71]

	t.Run("When applying the 0.16.7 server reporting migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_sets"))
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_items"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS server reporting projection tables", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_sets"))
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_items"))
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_items"))
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_reporting_sets"))
		})
	})

	return mig
}
