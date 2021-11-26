package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_4_2RemoveHistoryRemoteIDUnique(eng *testEngine, dialect string) {
	Convey("Given the 0.4.2 history unique constraint removal", func() {
		setupDatabaseUpTo(eng, ver0_4_2RemoveHistoryRemoteIDUnique{})
		So(doesIndexExist(eng.DB, dialect, "transfer_history",
			"UQE_transfer_history_histRemID"), ShouldBeTrue)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_4_2RemoveHistoryRemoteIDUnique{})
			So(err, ShouldBeNil)

			Convey("Then it should have removed the constraint (the index)", func() {
				So(doesIndexExist(eng.DB, dialect, "transfer_history",
					"UQE_transfer_history_histRemID"), ShouldBeFalse)
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_4_2RemoveHistoryRemoteIDUnique{})
				So(err, ShouldBeNil)

				Convey("Then it should have restored the constraint (the index)", func() {
					So(doesIndexExist(eng.DB, dialect, "transfer_history",
						"UQE_transfer_history_histRemID"), ShouldBeTrue)
				})
			})
		})
	})
}
