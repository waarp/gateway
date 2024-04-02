package migrations

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"code.waarp.fr/lib/migration"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

type testEngine struct {
	*migration.Engine
	DB      *sql.DB
	Dialect string
}

func (t *testEngine) Upgrade(changes ...Change) error {
	return t.Engine.Upgrade(changes) //nolint:wrapcheck //this is just for tests
}

func (t *testEngine) Downgrade(changes ...Change) error {
	return t.Engine.Downgrade(changes) //nolint:wrapcheck //this is just for tests
}

func (t *testEngine) NoError(tb testing.TB, query string, args ...any) {
	tb.Helper()

	if t.Dialect == PostgreSQL {
		for i := 1; strings.Contains(query, "?"); i++ {
			query = strings.Replace(query, "?", fmt.Sprintf("$%d", i), 1)
		}
	}

	_, err := t.DB.Exec(query, args...)
	require.NoError(tb, err)
}

func getSQLiteEngine(tb testing.TB) *testEngine {
	tb.Helper()

	logger := gwtesting.Logger(tb)
	db := gwtesting.SQLiteDatabase(tb)

	eng, err := migration.NewEngine(db, SQLite, logger, nil)
	require.NoError(tb, err)

	return &testEngine{Engine: eng, DB: db, Dialect: SQLite}
}

func TestSQLiteMigrations(t *testing.T) {
	t.Parallel()

	testMigrations(t, getSQLiteEngine(t))
}
