package migrations

import (
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

//nolint:stylecheck // function should contain the name of the version
func testVer0_4_2RemoveHistoryRemoteIdUnique(eng *migration.Engine, dialect string) {
	Convey("Given the 0.4.2 history unique constraint removal", func() {
		setupDatabaseUpTo(eng, ver0_4_2RemoveHistoryRemoteIDUnique{})
		So(doesIndexExist(eng.DB, dialect, "transfer_history",
			"UQE_transfer_history_histRemID"), ShouldBeTrue)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_4_2RemoveHistoryRemoteIDUnique{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have removed the constraint (the index)", func() {
				So(doesIndexExist(eng.DB, dialect, "transfer_history",
					"UQE_transfer_history_histRemID"), ShouldBeFalse)
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_4_2RemoveHistoryRemoteIDUnique{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have restored the constraint (the index)", func() {
					So(doesIndexExist(eng.DB, dialect, "transfer_history",
						"UQE_transfer_history_histRemID"), ShouldBeTrue)
				})
			})
		})
	})
}
