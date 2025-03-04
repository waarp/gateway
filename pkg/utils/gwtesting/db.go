package gwtesting

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func Database(tb testing.TB) *database.DB {
	tb.Helper()

	return dbtest.TestDatabase(tb)
}
