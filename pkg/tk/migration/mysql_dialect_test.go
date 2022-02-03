//go:build test_db_mysql
// +build test_db_mysql

package migration

import (
	"database/sql"
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type mysqlTestEngine struct{ *mySQLActions }

func (m *mysqlTestEngine) getTranslator() translator { return m.trad }

func testMySQLEngine(db *sql.DB) testEngine {
	return &mysqlTestEngine{&mySQLActions{
		standardSQL: &standardSQL{
			queryWriter: &queryWriter{db: db, writer: os.Stdout},
		},
		trad: &mysqlTranslator{},
	}}
}

func TestMySQLCreateTable(t *testing.T) {
	testSQLCreateTable(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLRenameTable(t *testing.T) {
	testSQLRenameTable(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLDropTable(t *testing.T) {
	testSQLDropTable(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLChangeColumnType(t *testing.T) {
	testSQLChangeColumnType(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLAddColumn(t *testing.T) {
	testSQLAddColumn(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLDropColumn(t *testing.T) {
	testSQLDropColumn(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}

func TestMySQLAddRow(t *testing.T) {
	testSQLAddRow(t, "MySQL", testhelpers.GetTestMySQLDB, testMySQLEngine)
}
