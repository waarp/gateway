//go:build test_db_postgresql
// +build test_db_postgresql

package migrations

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"

	"github.com/smartystreets/goconvey/convey"
	. "github.com/smartystreets/goconvey/convey"
)

func getPostgreEngine(c convey.C) *migration.Engine {
	db := testhelpers.GetTestPostgreDB(c)

	_, err := db.Exec(PostgresCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.PostgreSQL, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestPostgreSQLCreationScript(t *testing.T) {
	Convey("Given a PostgreSQL database", t, func(c C) {
		db := testhelpers.GetTestPostgreDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {
			Convey("When executing the script", func() {
				_, err := db.Exec(PostgresCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func TestPostgreSQLMigrations(t *testing.T) {
	const dbType = migration.PostgreSQL

	Convey("Given an un-migrated PostgreSQL database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}
