package migrations

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_4_0InitDatabase(t *testing.T, eng *testEngine) Change {
	mig := Migrations[0]

	isDBEmpty := func() bool {
		var row *sql.Row

		switch eng.Dialect {
		case SQLite:
			row = eng.DB.QueryRow(`SELECT name FROM sqlite_master WHERE 
                name<>'sqlite_sequence'`)
		case PostgreSQL:
			row = eng.DB.QueryRow(`SELECT tablename FROM pg_tables 
                 WHERE schemaname=current_schema()`)
		case MySQL:
			row = eng.DB.QueryRow(`SELECT table_name FROM information_schema.tables
        		WHERE table_schema = DATABASE()`)
		}

		var name string

		return errors.Is(row.Scan(&name), sql.ErrNoRows)
	}

	t.Run("When applying the 0.4.0 database initialization", func(t *testing.T) {
		assert.True(t, isDBEmpty(),
			"Before the migration, the database should be empty")

		require.NoError(t,
			eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have created the tables", func(t *testing.T) {
			for _, table := range initTableList() {
				assert.Truef(t,
					doesTableExist(t, eng.DB, eng.Dialect, table),
					"After the migration, the table %s should exist", table)
			}
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			assert.True(t, isDBEmpty(),
				"After reverting the migration, the database should be empty")
		})
	})

	return mig
}
