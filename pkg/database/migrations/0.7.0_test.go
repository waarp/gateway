package migrations

import . "github.com/smartystreets/goconvey/convey"

func testVer0_7_0AddLocalAgentEnabled(eng *testEngine) {
	Convey("Given the 0.6.0 server 'enable' addition", func() {
		setupDatabaseUpTo(eng, ver0_7_0AddLocalAgentEnabledColumn{})
		tableShouldNotHaveColumns(eng.DB, "local_agents", "enabled")

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0AddLocalAgentEnabledColumn{})
			So(err, ShouldBeNil)

			Convey("Then it should have added the column", func() {
				tableShouldHaveColumns(eng.DB, "local_agents", "enabled")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0AddLocalAgentEnabledColumn{})
				So(err, ShouldBeNil)

				Convey("Then it should have dropped the column", func() {
					tableShouldNotHaveColumns(eng.DB, "local_agents", "enabled")
				})
			})
		})
	})
}
