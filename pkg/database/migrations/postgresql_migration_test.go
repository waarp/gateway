//go:build test_db_postgresql

package migrations

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func getPostgreEngine(c convey.C) *testEngine {
	logger := testhelpers.TestLogger(c, "test_postgre_engine")
	db := testhelpers.GetTestPostgreDB(c)

	eng, err := migration.NewEngine(db, migration.PostgreSQL, logger, nil)
	So(err, ShouldBeNil)

	return &testEngine{Engine: eng, DB: db}
}

func TestPostgreSQLMigrations(t *testing.T) {
	Convey("Given an un-migrated PostgreSQL database engine", t, func(c C) {
		testMigrations(getPostgreEngine(c), migration.PostgreSQL)
	})
}
