// Package migtest contains utilities for testing database migrations.
package migtest

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

// SQLiteDatabase returns a SQLite database connection for testing.
func SQLiteDatabase(tb testing.TB) *sql.DB {
	tb.Helper()

	path := filepath.Join(tb.TempDir(), "test.db")
	values := url.Values{}

	values.Set("mode", "rwc")
	values.Set("cache", "shared")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout(10000)")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "synchronous(NORMAL)")

	dsn := fmt.Sprintf("file:%s?%s", path, values.Encode())

	db, err := sql.Open("sqlite", dsn)
	require.NoError(tb, err)

	tb.Cleanup(func() {
		require.NoError(tb, db.Close())
	})

	return db
}

// PostgreSQLDatabase returns a PostgreSQL database connection for testing.
func PostgreSQLDatabase(tb testing.TB) *sql.DB {
	tb.Helper()

	db, openErr := sql.Open("pgx", "user='postgres' password='postgres' "+
		"host='localhost' port='5432' dbname='waarp_gateway_test' sslmode='disable' "+
		"statement_cache_capacity=0 default_query_exec_mode=describe_exec")
	require.NoError(tb, openErr)

	_, createErr := db.Exec("CREATE SCHEMA IF NOT EXISTS public")
	require.NoError(tb, createErr)

	_, tzErr := db.Exec("ALTER DATABASE waarp_gateway_test SET TIME ZONE 'UTC'")
	require.NoError(tb, tzErr)

	tb.Cleanup(func() {
		_, dropErr := db.Exec("DROP SCHEMA public CASCADE")
		require.NoError(tb, dropErr)

		_, recreatErr := db.Exec("CREATE SCHEMA public")
		require.NoError(tb, recreatErr)

		require.NoError(tb, db.Close())
	})

	return db
}

// MySQLDatabase returns a MySQL database connection for testing.
func MySQLDatabase(tb testing.TB) *sql.DB {
	tb.Helper()

	conf := mysql.NewConfig()
	conf.User = "mysql"
	conf.Passwd = "mysql"
	conf.DBName = "waarp_gateway_test"
	conf.Addr = "localhost:3306"
	conf.ParseTime = true

	db, openErr := sql.Open("mysql", conf.FormatDSN())
	require.NoError(tb, openErr)

	tb.Cleanup(func() {
		_, dropErr := db.Exec("DROP DATABASE waarp_gateway_test")
		require.NoError(tb, dropErr)

		_, recreatErr := db.Exec("CREATE DATABASE waarp_gateway_test")
		require.NoError(tb, recreatErr)

		require.NoError(tb, db.Close())
	})

	return db
}
