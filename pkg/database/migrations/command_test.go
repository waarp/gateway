package migrations

import (
	"testing"

	"code.waarp.fr/lib/migration"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestDoMigration(t *testing.T) {
	Convey("Given an test database", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_do_migration")
		db := getSQLiteEngine(c).DB
		var curr string

		So(DoMigration(db, logger, "0.4.0", migration.SQLite, nil), ShouldBeNil)

		So(db.QueryRow(`SELECT current FROM version`).Scan(&curr), ShouldBeNil)
		So(curr, ShouldEqual, "0.4.0")

		Convey("When migrating up to a version", func() {
			err := DoMigration(db, logger, "0.4.2", migration.SQLite, nil)
			So(err, ShouldBeNil)

			Convey("Then it should have executed all the change scripts "+
				"up to the given version", func() {
				So(doesIndexExist(db, SQLite, "transfer_history",
					"UQE_transfer_history_histRemID"), ShouldBeFalse)

				So(db.QueryRow(`SELECT current FROM version`).Scan(&curr), ShouldBeNil)
				So(curr, ShouldEqual, "0.4.2")

				Convey("When migrating down to a version", func() {
					err := DoMigration(db, logger, "0.4.0", migration.SQLite, nil)
					So(err, ShouldBeNil)

					Convey("Then it should have executed all the change scripts "+
						"down to the given version", func() {
						So(doesIndexExist(db, SQLite, "transfer_history",
							"UQE_transfer_history_histRemID"), ShouldBeTrue)

						So(db.QueryRow(`SELECT current FROM version`).Scan(&curr), ShouldBeNil)
						So(curr, ShouldEqual, "0.4.0")
					})
				})
			})
		})

		Convey("When migrating between versions with the same index", func() {
			const testTarget = "test_target"

			VersionsMap[testTarget] = VersionsMap["0.4.0"]

			So(DoMigration(db, logger, testTarget, migration.SQLite, nil), ShouldBeNil)

			Convey("Then it should have changed the database version", func() {
				So(db.QueryRow(`SELECT current FROM version`).Scan(&curr), ShouldBeNil)
				So(curr, ShouldEqual, testTarget)
			})
		})
	})
}
