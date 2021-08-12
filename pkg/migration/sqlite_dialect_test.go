package migration

import (
	"database/sql"
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"

	. "github.com/smartystreets/goconvey/convey"
)

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
	testSQLCreateTable(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteRenameTable(t *testing.T) {
	testSQLRenameTable(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteDropTable(t *testing.T) {
	testSQLDropTable(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteChangeColumnType(t *testing.T) {
	testSQLChangeColumnType(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteAddColumn(t *testing.T) {
	testSQLAddColumn(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteDropColumn(t *testing.T) {
	testSQLDropColumn(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}

func TestSQLiteAddRow(t *testing.T) {
	testSQLAddRow(t, "SQLite", testhelpers.GetTestSqliteDB, testSQLiteEngine)
}
