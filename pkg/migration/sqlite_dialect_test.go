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
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc", file))
	So(err, ShouldBeNil)

	Reset(func() {
		So(db.Close(), ShouldBeNil)
		So(os.Remove(file), ShouldBeNil)
	})
	return db
}

func testSqliteEngine(db *sql.DB) Dialect {
	return newSqliteEngine(&queryWriter{db: db, writer: os.Stdout})
}

func TestSqliteCreateTable(t *testing.T) {
	testSQLCreateTable(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSqliteRenameTable(t *testing.T) {
	testSQLRenameTable(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSqliteDropTable(t *testing.T) {
	testSQLDropTable(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSqliteRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSqliteAddColumn(t *testing.T) {
	testSQLAddColumn(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSQLiteDropColumn(t *testing.T) {
	testSQLDropColumn(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}

func TestSQLiteAddRow(t *testing.T) {
	testSQLAddRow(t, "SQLite", getTestSqliteDB, testSqliteEngine)
}
