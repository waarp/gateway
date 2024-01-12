package migrations

import (
	"database/sql"
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
