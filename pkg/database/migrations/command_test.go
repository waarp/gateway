package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations/migtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestDoMigration(t *testing.T) {
	t.Parallel()

	t.Run("Given an test database", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := migtest.SQLiteDatabase(t)
		var curr string

		require.NoError(t, DoMigration(db, logger, "0.4.0", SQLite, nil))

		require.NoError(t, db.QueryRow(`SELECT current FROM version`).Scan(&curr))
		require.Equal(t, "0.4.0", curr)

		t.Run("When migrating up to a version", func(t *testing.T) {
			require.NoError(t, DoMigration(db, logger, "0.4.2", SQLite, nil))

			t.Run("Then it should have executed all the change scripts "+
				"up to the given version", func(t *testing.T) {
				assert.False(t, doesIndexExist(t, db, SQLite,
					"transfer_history", "UQE_transfer_history_histRemID"))

				require.NoError(t, db.QueryRow(`SELECT current FROM version`).Scan(&curr))
				assert.Equal(t, "0.4.2", curr)

				t.Run("When migrating down to a version", func(t *testing.T) {
					require.NoError(t, DoMigration(db, logger, "0.4.0", SQLite, nil))

					t.Run("Then it should have executed all the change scripts "+
						"down to the given version", func(t *testing.T) {
						assert.True(t, doesIndexExist(t, db, SQLite,
							"transfer_history", "UQE_transfer_history_histRemID"))

						require.NoError(t, db.QueryRow(`SELECT current FROM version`).Scan(&curr))
						assert.Equal(t, "0.4.0", curr)
					})
				})
			})
		})

		t.Run("When migrating between versions with the same index", func(t *testing.T) {
			const testTarget = "test_target"
			VersionsMap[testTarget] = VersionsMap["0.4.0"]

			t.Cleanup(func() {
				delete(VersionsMap, testTarget)
			})

			require.NoError(t, DoMigration(db, logger, testTarget, SQLite, nil))

			t.Run("Then it should have changed the database version", func(t *testing.T) {
				require.NoError(t, db.QueryRow(`SELECT current FROM version`).Scan(&curr))
				assert.Equal(t, testTarget, curr)
			})
		})
	})
}
