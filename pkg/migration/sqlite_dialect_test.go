package migration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func getTestSqliteDB() *sql.DB {
	file := tempFilename()
	//db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc", file))
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc", file))
	So(err, ShouldBeNil)

	Reset(func() {
		So(db.Close(), ShouldBeNil)
		So(os.Remove(file), ShouldBeNil)
	})
	return db
}

func testSQLiteEngine(db *sql.DB) testEngine {
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	So(err, ShouldBeNil)
	return &sqliteDialect{
		standardSQL: &standardSQL{
			queryWriter: &queryWriter{db: db, writer: os.Stdout},
		},
	}
}

func TestSQLiteCreateTable(t *testing.T) {
	testSQLCreateTable(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteRenameTable(t *testing.T) {
	testSQLRenameTable(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteDropTable(t *testing.T) {
	testSQLDropTable(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteChangeColumnType(t *testing.T) {
	testSQLChangeColumnType(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteAddColumn(t *testing.T) {
	testSQLAddColumn(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteDropColumn(t *testing.T) {
	testSQLDropColumn(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteAddRow(t *testing.T) {
	testSQLAddRow(t, "SQLite", getTestSqliteDB, testSQLiteEngine)
}
