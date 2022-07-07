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

	_, err := db.Exec(PostgresCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.PostgreSQL, logger, nil)
	So(err, ShouldBeNil)

	return &testEngine{Engine: eng, DB: db}
}

func TestPostgreSQLCreationScript(t *testing.T) {
	Convey("Given a PostgreSQL database", t, func(c C) {
		db := testhelpers.GetTestPostgreDB(c)

		Convey("Given the change to initialize version 0.0.0 of the database", func() {
			Convey("When executing the change", func() {
				_, err := db.Exec(PostgresCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func TestPostgreSQLMigrations(t *testing.T) {
	Convey("Given an un-migrated PostgreSQL database engine", t, func(c C) {
		testMigrations(getPostgreEngine(c), migration.PostgreSQL)
	})
}
