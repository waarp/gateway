//go:build test_full || test_db_postgresql
// +build test_full test_db_postgresql

package migration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type postgreTestEngine struct{ *postgreActions }

func (p *postgreActions) getTranslator() translator { return p.trad }

func testPostgreEngine(db *sql.DB) testEngine {
	return &postgreTestEngine{&postgreActions{
		standardSQL: &standardSQL{
			queryWriter: &queryWriter{db: db, writer: os.Stdout},
		},
		trad: &postgreTranslator{},
	}}
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
