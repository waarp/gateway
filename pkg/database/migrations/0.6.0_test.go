package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_6_0AddTransferInfoIsHistory(eng *testEngine) {
	Convey("Given the 0.6.0 transfer info 'is_history' addition", func() {
		setupDatabaseUpTo(eng, ver0_6_0AddTransferInfoIsHistory{})
		tableShouldNotHaveColumns(eng.DB, "transfer_info", "is_history")

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_6_0AddTransferInfoIsHistory{})
			So(err, ShouldBeNil)

			Convey("Then it should have added the column", func() {
				tableShouldHaveColumns(eng.DB, "transfer_info", "is_history")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_6_0AddTransferInfoIsHistory{})
				So(err, ShouldBeNil)

				Convey("Then it should have dropped the column", func() {
					tableShouldNotHaveColumns(eng.DB, "transfer_info", "is_history")
				})
			})
		})
	})
}
