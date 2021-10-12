package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func getSQLiteEngine(c C) *migration.Engine {
	db := testhelpers.GetTestSqliteDB(c)

	_, err := db.Exec(SqliteCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.SQLite, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestSQLiteCreationScript(t *testing.T) {
	Convey("Given a SQLite database", t, func(c C) {
		db := testhelpers.GetTestSqliteDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {
			Convey("When executing the script", func() {
				_, err := db.Exec(SqliteCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func TestSQLiteMigrations(t *testing.T) {
	const dbType = migration.SQLite

	Convey("Given an un-migrated SQLite database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}
