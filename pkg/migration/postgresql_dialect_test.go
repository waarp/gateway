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
		rows, err := db.Query(`SELECT 'DROP TABLE IF EXISTS "' || tablename || '" CASCADE;' 
			FROM pg_tables WHERE schemaname = 'public'`)
		So(err, ShouldBeNil)

		for rows.Next() {
			var cmd string
			So(rows.Scan(&cmd), ShouldBeNil)
			_, err := db.Exec(cmd)
			So(err, ShouldBeNil)
		}

		So(rows.Close(), ShouldBeNil)
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
