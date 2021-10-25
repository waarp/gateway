package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

func TestDoMigration(t *testing.T) {
	Convey("Given an origin test database", t, func(c C) {
		db := getSQLiteEngine(c).DB

		Convey("When migrating up to a version", func() {
			err := doMigration(db, "0.4.1", migration.SQLite, nil)
			So(err, ShouldBeNil)

			Convey("Then it should have executed all the migrations scripts "+
				"up to the given version", func() {
				row := db.QueryRow(`SELECT current FROM version`)

				var curr string
				So(row.Scan(&curr), ShouldBeNil)

				So(curr, ShouldEqual, "0.4.1")

				Convey("When migrating down to a version", func() {
					err := doMigration(db, "0.4.0", migration.SQLite, nil)
					So(err, ShouldBeNil)

					Convey("Then it should have executed all the migrations scripts "+
						"down to the given version", func() {
						row := db.QueryRow(`SELECT current FROM version`)

						var curr string
						So(row.Scan(&curr), ShouldBeNil)

						So(curr, ShouldEqual, "0.4.0")
					})
				})
			})
		})
	})
}
