package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_16_6AddEbicsServerContractProjectionTables(t *testing.T, eng *testEngine) Change {
	mig := Migrations[70]

	t.Run("When applying the 0.16.6 server contract migration", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_sets"))
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_items"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should add the EBICS server contract projection tables", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_sets"))
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_items"))
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_items"))
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "ebics_server_contract_sets"))
		})
	})

	return mig
}
