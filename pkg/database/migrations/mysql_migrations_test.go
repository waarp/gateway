//go:build test_db_mysql

package migrations

import (
	"testing"

	"code.waarp.fr/lib/migration"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations/migtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func getMySQLEngine(tb testing.TB) *testEngine {
	logger := testhelpers.GetTestLogger(tb)
	db := migtest.MySQLDatabase(tb)

	eng, err := migration.NewEngine(db, MySQL, logger, nil)
	require.NoError(tb, err)

	return &testEngine{Engine: eng, DB: db, Dialect: MySQL}
}

func TestMySQLMigrations(t *testing.T) {
	t.Parallel()

	testMigrations(t, getMySQLEngine(t))
}
