// +build test_full test_db_mysql

package migration

import (
	"database/sql"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	. "github.com/smartystreets/goconvey/convey"
)

func getTestMySQLDB() *sql.DB {
	conf := mysql.NewConfig()
	conf.User = "root"
	conf.DBName = "waarp_gateway_test"
	conf.Addr = "localhost:3306"
	db, err := sql.Open("mysql", conf.FormatDSN())
	So(err, ShouldBeNil)

	Reset(func() {
		_, err := db.Exec("DROP DATABASE waarp_gateway_test")
		So(err, ShouldBeNil)
		_, err = db.Exec("CREATE DATABASE waarp_gateway_test")
		So(err, ShouldBeNil)
		So(db.Close(), ShouldBeNil)
	})
	return db
}

func testMySQLEngine(db *sql.DB) testEngine {
	return &mySQLDialect{
		standardSQL: &standardSQL{
			queryWriter: &queryWriter{db: db, writer: os.Stdout},
		},
	}
}

func TestMySQLCreateTable(t *testing.T) {
	testSQLCreateTable(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLRenameTable(t *testing.T) {
	testSQLRenameTable(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLDropTable(t *testing.T) {
	testSQLDropTable(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLChangeColumnType(t *testing.T) {
	testSQLChangeColumnType(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLAddColumn(t *testing.T) {
	testSQLAddColumn(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLDropColumn(t *testing.T) {
	testSQLDropColumn(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}

func TestMySQLAddRow(t *testing.T) {
	testSQLAddRow(t, "MySQL", getTestMySQLDB, testMySQLEngine)
}
