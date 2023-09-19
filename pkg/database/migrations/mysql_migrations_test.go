//go:build test_db_mysql

package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func getMySQLEngine(c C) *testEngine {
	logger := testhelpers.TestLogger(c, "test_mysql_engine")
	db := testhelpers.GetTestMySQLDB(c)

	eng, err := migration.NewEngine(db, migration.MySQL, logger, nil)
	So(err, ShouldBeNil)

	return &testEngine{Engine: eng, DB: db}
}

func TestMySQLMigrations(t *testing.T) {
	t.Parallel()

	Convey("Given an un-migrated MySQL database engine", t, func(c C) {
		testMigrations(getMySQLEngine(c), migration.MySQL)
	})
}
