package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"
	sqliteLib "modernc.org/sqlite/lib"
)

func doesIndexExist(tb testing.TB, db *sql.DB, dialect, table, index string) bool {
	tb.Helper()

	var (
		rows *sql.Rows
		err  error
	)

	switch dialect {
	case SQLite:
		rows, err = db.Query("SELECT name FROM sqlite_master WHERE type=? AND name=?", "index", index)
	case PostgreSQL:
		rows, err = db.Query("SELECT indexname FROM pg_indexes WHERE indexname=$1", index)
	case MySQL:
		rows, err = db.Query("SHOW INDEX FROM "+table+" WHERE Key_name=?", index)
	default:
		panic(fmt.Sprintf("unknown database engine '%s'", dialect))
	}

	require.NoError(tb, err)
	require.NoError(tb, rows.Err())

	defer rows.Close()

	return rows.Next()
}

func doesTableExist(tb testing.TB, db *sql.DB, dbType, table string) bool {
	tb.Helper()

	var (
		row  *sql.Row
		name string
	)

	switch dbType {
	case SQLite:
		row = db.QueryRow(`SELECT name FROM sqlite_master WHERE
			type='table' AND name=?`, table)
	case PostgreSQL:
		row = db.QueryRow(`SELECT tablename FROM pg_tables WHERE tablename=$1`, table)
	case MySQL:
		row = db.QueryRow(fmt.Sprintf(`SHOW TABLES LIKE '%s'`, table))
	default:
		panic(fmt.Sprintf("unknown database type: %s", dbType))
	}

	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}

		require.NoError(tb, err)
	}

	return true
}

func getColumns(tb testing.TB, db *sql.DB, table string) []string {
	tb.Helper()

	rows, err := db.Query("SELECT * FROM " + table)
	require.NoError(tb, err)
	require.NoError(tb, rows.Err())

	defer rows.Close()

	names, err := rows.Columns()
	require.NoError(tb, err)

	return names
}

func tableShouldHaveColumns(tb testing.TB, db *sql.DB, table string, cols ...string) {
	tb.Helper()

	names := getColumns(tb, db, table)

	for _, col := range cols {
		assert.Containsf(tb, names, col,
			"Table %q should have column %q (but it didn't)", table, col)
	}
}

func tableShouldNotHaveColumns(tb testing.TB, db *sql.DB, table string, cols ...string) {
	tb.Helper()

	names := getColumns(tb, db, table)

	for _, col := range cols {
		assert.NotContainsf(tb, names, col,
			"Table %q should not have column %q (but it did)", table, col)
	}
}

func shouldBeUniqueViolationError(tb testing.TB, err error) {
	tb.Helper()

	var (
		sqlErr *sqlite.Error
		pgErr  *pgconn.PgError
		myErr  *mysql.MySQLError
	)

	switch {
	case errors.As(err, &sqlErr):
		assert.Equal(tb, sqliteLib.SQLITE_CONSTRAINT_UNIQUE, sqlErr.Code())
	case errors.As(err, &pgErr):
		assert.Equal(tb, pgerrcode.UniqueViolation, pgErr.Code)
	case errors.As(err, &myErr):
		assert.Equal(tb, uint16(1062), myErr.Number)
	default:
		tb.Errorf("unexpected error type: %T", err)
	}
}

func shouldBeTableNotExist(tb testing.TB, err error) {
	tb.Helper()

	var (
		sqlErr *sqlite.Error
		pgErr  *pgconn.PgError
		myErr  *mysql.MySQLError
	)

	switch {
	case errors.As(err, &sqlErr):
		assert.Contains(tb, sqlErr.Error(), `SQL logic error: no such table:`)
	case errors.As(err, &pgErr):
		assert.Equal(tb, pgerrcode.UndefinedTable, pgErr.Code)
	case errors.As(err, &myErr):
		assert.Equal(tb, uint16(1146), myErr.Number)
	default:
		tb.Errorf("unexpected error type: %T", err)
	}
}
