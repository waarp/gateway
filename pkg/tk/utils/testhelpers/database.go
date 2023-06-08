package testhelpers

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/smartystreets/goconvey/convey"
	_ "modernc.org/sqlite"
)

// GetTestSqliteDBNoReset returns a *sql.DB pointing to a test SQLite database.
// The database will not be automatically reset at the end of the test.
func GetTestSqliteDBNoReset(c convey.C) (*sql.DB, string) {
	f, err := os.CreateTemp(os.TempDir(), "test_migration_database_*.db")
	c.So(err, convey.ShouldBeNil)
	c.So(f.Close(), convey.ShouldBeNil)
	c.So(os.Remove(f.Name()), convey.ShouldBeNil)

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rwc&cache=shared", f.Name()))
	c.So(err, convey.ShouldBeNil)

	return db, f.Name()
}

// GetTestSqliteDB returns a *sql.DB pointing to a test SQLite database.
func GetTestSqliteDB(c convey.C) *sql.DB {
	db, file := GetTestSqliteDBNoReset(c)

	c.Reset(func() {
		c.So(db.Close(), convey.ShouldBeNil)
		c.So(os.Remove(file), convey.ShouldBeNil)
	})

	return db
}

// GetTestPostgreDBNoReset returns a *sql.DB pointing to a test PostgreSQL database.
// The database will not be automatically reset at the end of the test.
func GetTestPostgreDBNoReset(c convey.C) *sql.DB {
	db, err := sql.Open("pgx", "user='postgres' host='localhost' port='5432' "+
		"dbname='waarp_gateway_test' sslmode='disable' statement_cache_capacity=0")
	c.So(err, convey.ShouldBeNil)

	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS public")
	c.So(err, convey.ShouldBeNil)

	return db
}

// GetTestPostgreDB returns a *sql.DB pointing to a test PostgreSQL database.
// The database must be named 'waarp_gateway_test', run on the default 5432 port,
// and use the default 'postgres' user.
func GetTestPostgreDB(c convey.C) *sql.DB {
	db := GetTestPostgreDBNoReset(c)

	c.Reset(func() {
		_, err := db.Exec("DROP SCHEMA public CASCADE")
		c.So(err, convey.ShouldBeNil)

		_, err = db.Exec("CREATE SCHEMA public")
		c.So(err, convey.ShouldBeNil)
		c.So(db.Close(), convey.ShouldBeNil)
	})

	return db
}

// GetTestMySQLDBNoReset returns a *sql.DB pointing to a test MySQL database.
// The database will not be automatically reset at the end of the test.
func GetTestMySQLDBNoReset(c convey.C) *sql.DB {
	conf := mysql.NewConfig()
	conf.User = "root"
	conf.DBName = "waarp_gateway_test"
	conf.Addr = "localhost:3306"
	conf.ParseTime = true

	db, err := sql.Open("mysql", conf.FormatDSN())
	c.So(err, convey.ShouldBeNil)

	return db
}

// GetTestMySQLDB returns a *sql.DB pointing to a test MySQL database.
// The database must be named 'waarp_gateway_test', run on the default 3306 port,
// and use the default 'root' user.
func GetTestMySQLDB(c convey.C) *sql.DB {
	db := GetTestMySQLDBNoReset(c)

	c.Reset(func() {
		_, err := db.Exec("DROP DATABASE waarp_gateway_test")
		c.So(err, convey.ShouldBeNil)

		_, err = db.Exec("CREATE DATABASE waarp_gateway_test")
		c.So(err, convey.ShouldBeNil)
		c.So(db.Close(), convey.ShouldBeNil)
	})

	return db
}
