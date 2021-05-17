// +build test_full test_db_postgresql

package migration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v4"

	. "github.com/smartystreets/goconvey/convey"
)

func getTestPostgreDB() *sql.DB {
	db, err := sql.Open("pgx", "user='postgres' host='localhost' port='5432' "+
		"dbname='waarp_gateway_test' sslmode='disable'")
	So(err, ShouldBeNil)

	Reset(func() {
		_, err := db.Exec("DROP SCHEMA public CASCADE")
		So(err, ShouldBeNil)
		_, err = db.Exec("CREATE SCHEMA public")
		So(err, ShouldBeNil)
		So(db.Close(), ShouldBeNil)
	})
	return db
}

func testPostgreEngine(db *sql.DB) Dialect {
	return newPostgreEngine(&queryWriter{db: db, writer: os.Stdout})
}

func TestPostgreCreateTable(t *testing.T) {
	testSQLCreateTable(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreRenameTable(t *testing.T) {
	testSQLRenameTable(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreDropTable(t *testing.T) {
	testSQLDropTable(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreAddColumn(t *testing.T) {
	testSQLAddColumn(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreDropColumn(t *testing.T) {
	testSQLDropColumn(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}

func TestPostgreAddRow(t *testing.T) {
	testSQLAddRow(t, "PostgreSQL", getTestPostgreDB, testPostgreEngine)
}
