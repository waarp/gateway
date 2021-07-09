// +build test_full test_db_postgresql

package migration

import (
	"database/sql"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	_ "github.com/jackc/pgx/v4"
)

func testPostgreEngine(db *sql.DB) testEngine {
	return &postgreDialect{
		standardSQL: &standardSQL{
			queryWriter: &queryWriter{db: db, writer: os.Stdout},
		},
	}
}

func TestPostgreCreateTable(t *testing.T) {
	testSQLCreateTable(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreRenameTable(t *testing.T) {
	testSQLRenameTable(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreDropTable(t *testing.T) {
	testSQLDropTable(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreRenameColumn(t *testing.T) {
	testSQLRenameColumn(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreChangeColumnType(t *testing.T) {
	testSQLChangeColumnType(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreAddColumn(t *testing.T) {
	testSQLAddColumn(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreDropColumn(t *testing.T) {
	testSQLDropColumn(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}

func TestPostgreAddRow(t *testing.T) {
	testSQLAddRow(t, "PostgreSQL", testhelpers.GetTestPostgreDB, testPostgreEngine)
}
