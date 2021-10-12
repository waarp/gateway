package testhelpers

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/smartystreets/goconvey/convey"
)

// GetTestSqliteDB returns a *sql.DB pointing to an in-memory test SQLite database.
func GetTestSqliteDB(c convey.C) *sql.DB {
	f, err := ioutil.TempFile(os.TempDir(), "test_migration_database_*.db")
	c.So(err, convey.ShouldBeNil)
	c.So(f.Close(), convey.ShouldBeNil)
	c.So(os.Remove(f.Name()), convey.ShouldBeNil)

	file := f.Name()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc", file))

	c.So(err, convey.ShouldBeNil)

	c.Reset(func() {
		c.So(db.Close(), convey.ShouldBeNil)
		c.So(os.Remove(file), convey.ShouldBeNil)
	})

	return db
}

// GetTestPostgreDB returns a *sql.DB pointing to a test PostgreSQL database.
// The database must be named 'waarp_gateway_test', run on the default 5432 port,
// and use the default 'postgres' user.
func GetTestPostgreDB(c convey.C) *sql.DB {
	db, err := sql.Open("pgx", "user='postgres' host='localhost' port='5432' "+
		"dbname='waarp_gateway_test' sslmode='disable' statement_cache_capacity=0")
	c.So(err, convey.ShouldBeNil)

	c.Reset(func() {
		_, err := db.Exec("DROP SCHEMA public CASCADE")
		c.So(err, convey.ShouldBeNil)

		_, err = db.Exec("CREATE SCHEMA public")
		c.So(err, convey.ShouldBeNil)
		c.So(db.Close(), convey.ShouldBeNil)
	})

	return db
}

// GetTestMySQLDB returns a *sql.DB pointing to a test MySQL database.
// The database must be named 'waarp_gateway_test', run on the default 3306 port,
// and use the default 'root' user.
func GetTestMySQLDB(c convey.C) *sql.DB {
	conf := mysql.NewConfig()
	conf.User = "root"
	conf.DBName = "waarp_gateway_test"
	conf.Addr = "localhost:3306"

	db, err := sql.Open("mysql", conf.FormatDSN())
	c.So(err, convey.ShouldBeNil)

	c.Reset(func() {
		_, err := db.Exec("DROP DATABASE waarp_gateway_test")
		c.So(err, convey.ShouldBeNil)
		_, err = db.Exec("CREATE DATABASE waarp_gateway_test")
		c.So(err, convey.ShouldBeNil)
		c.So(db.Close(), convey.ShouldBeNil)
	})

	return db
}
