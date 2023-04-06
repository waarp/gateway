package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_7_5SplitR66TLS(eng *testEngine) {
	Convey("Given the 0.7.5 R66 agents split", func() {
		mig := ver0_7_5SplitR66TLS{}
		setupDatabaseUpTo(eng, mig)

		// ### Local agents ###
		_, err := eng.DB.Exec(
			`INSERT INTO local_agents(id,owner,name,protocol,address,proto_config) 
			VALUES (1,'waarp_gw','gw_r66_1','r66','localhost:1','{"isTLS":true}'),
			       (2,'waarp_gw','gw_r66_2','r66','localhost:2','{"isTLS":false}')`)
		So(err, ShouldBeNil)

		// ### Remote agents ###
		_, err = eng.DB.Exec(
			`INSERT INTO remote_agents(id,name,protocol,address,proto_config) 
			VALUES (3,'waarp_r66_1','r66','localhost:3','{"isTLS":true}'),
			       (4,'waarp_r66_2','r66','localhost:4','{"isTLS":false}')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have split the R66 agents", func() {
				rows, err := eng.DB.Query(`
					SELECT id,protocol,proto_config FROM local_agents UNION ALL 
					SELECT id,protocol,proto_config FROM remote_agents ORDER BY id`)
				So(err, ShouldBeNil)

				defer rows.Close()

				nextRowShouldBe(rows, 1, "r66-tls", `{"isTLS":true}`)
				nextRowShouldBe(rows, 2, "r66", `{"isTLS":false}`)
				nextRowShouldBe(rows, 3, "r66-tls", `{"isTLS":true}`)
				nextRowShouldBe(rows, 4, "r66", `{"isTLS":false}`)

				So(rows.Next(), ShouldBeFalse)
				So(rows.Err(), ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				// Adding new R66-TLS agents without "isTLS" to test if these
				// cases are handled properly when migrating down.
				_, err := eng.DB.Exec(
					`INSERT INTO remote_agents(id,name,protocol,address,proto_config) 
						VALUES (5,'new_waarp_r66-tls','r66-tls','localhost:5','{}'),
						       (6,'new_waarp_r66','r66','localhost:6','{}')`)
				So(err, ShouldBeNil)

				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have restored the R66 agents", func() {
					rows, err := eng.DB.Query(`
					SELECT id,protocol,proto_config FROM local_agents UNION ALL 
					SELECT id,protocol,proto_config FROM remote_agents ORDER BY id`)
					So(err, ShouldBeNil)

					defer rows.Close()

					nextRowShouldBe(rows, 1, "r66", `{"isTLS":true}`)
					nextRowShouldBe(rows, 2, "r66", `{"isTLS":false}`)
					nextRowShouldBe(rows, 3, "r66", `{"isTLS":true}`)
					nextRowShouldBe(rows, 4, "r66", `{"isTLS":false}`)
					nextRowShouldBe(rows, 5, "r66", `{"isTLS":true}`)
					nextRowShouldBe(rows, 6, "r66", `{"isTLS":false}`)

					So(rows.Next(), ShouldBeFalse)
					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}
