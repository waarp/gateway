//go:build test_db_postgresql

package migrations

import (
	"testing"

	"code.waarp.fr/lib/migration"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations/migtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func getPostgreEngine(tb testing.TB) *testEngine {
	logger := testhelpers.GetTestLogger(tb)
	db := migtest.PostgreSQLDatabase(tb)

	eng, err := migration.NewEngine(db, PostgreSQL, logger, nil)
	require.NoError(tb, err)

	return &testEngine{Engine: eng, DB: db, Dialect: PostgreSQL}
}

func TestPostgreSQLMigrations(t *testing.T) {
	t.Parallel()

	testMigrations(t, getPostgreEngine(t))
}
