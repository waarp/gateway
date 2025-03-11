package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_12_0AddCryptoKeysOwner(t *testing.T, eng *testEngine) Change {
	mig := Migrations[58]

	t.Run("When applying the 0.12.1 crypto keys owner addition", func(t *testing.T) {
		require.True(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))

		// setup
		eng.NoError(t, `INSERT INTO users(owner, username) VALUES
			('owner1', 'user1a'), ('owner1', 'user1b'), ('owner2', 'user2')`)

		keyCol := quoteDial(eng.Dialect, "key")
		eng.NoError(t, `INSERT INTO crypto_keys(id, name, type, `+keyCol+`) VALUES
			(1, 'key1', 'type1', 'value1'), (2, 'key2', 'type2', 'value2')`,
		)

		// Pgsql does not increment the sequence when IDs are inserted manually,
		// so we have to manually increment the sequences to keep the test
		// consistent with other databases.
		if eng.Dialect == PostgreSQL {
			eng.NoError(t, `SELECT setval('crypto_keys_id_seq', 2)`)
		}

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM users`)
			eng.NoError(t, `DELETE FROM crypto_keys`)
		})

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "crypto_keys",
				"id", "owner", "name", "type", "value")
		})

		t.Run("Then it should have duplicated the keys", func(t *testing.T) {
			rows, queryErr := eng.DB.Query(`SELECT id, owner, name, type, value
				FROM crypto_keys ORDER BY id`)
			require.NoError(t, queryErr)

			defer rows.Close()

			var (
				id                      int
				owner, name, typ, value string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &typ, &value))
			assert.Equal(t, 3, id)
			assert.Equal(t, "owner1", owner)
			assert.Equal(t, "key1", name)
			assert.Equal(t, "type1", typ)
			assert.Equal(t, "value1", value)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &typ, &value))
			assert.Equal(t, 4, id)
			assert.Equal(t, "owner1", owner)
			assert.Equal(t, "key2", name)
			assert.Equal(t, "type2", typ)
			assert.Equal(t, "value2", value)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &typ, &value))
			assert.Equal(t, 5, id)
			assert.Equal(t, "owner2", owner)
			assert.Equal(t, "key1", name)
			assert.Equal(t, "type1", typ)
			assert.Equal(t, "value1", value)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &typ, &value))
			assert.Equal(t, 6, id)
			assert.Equal(t, "owner2", owner)
			assert.Equal(t, "key2", name)
			assert.Equal(t, "type2", typ)
			assert.Equal(t, "value2", value)

			assert.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "crypto_keys", "owner")
			})

			t.Run("Then it should have removed duplicated keys", func(t *testing.T) {
				rows, queryErr := eng.DB.Query(`SELECT id, name, type, ` + keyCol + `
					FROM crypto_keys ORDER BY id`)
				require.NoError(t, queryErr)

				defer rows.Close()

				var (
					id             int
					name, typ, key string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &typ, &key))
				assert.Equal(t, 3, id)
				assert.Equal(t, "key1", name)
				assert.Equal(t, "type1", typ)
				assert.Equal(t, "value1", key)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &typ, &key))
				assert.Equal(t, 4, id)
				assert.Equal(t, "key2", name)
				assert.Equal(t, "type2", typ)
				assert.Equal(t, "value2", key)

				assert.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}
