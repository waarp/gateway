package migrations

import . "github.com/smartystreets/goconvey/convey"

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
					SELECT id,protocol,proto_config FROM remote_agents`)
				So(err, ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						id          int64
						proto, conf string
					)

					So(rows.Scan(&id, &proto, &conf), ShouldBeNil)

					if id == 1 || id == 3 {
						So(conf, ShouldEqual, `{"isTLS":true}`)
						So(proto, ShouldEqual, "r66-tls")
					} else {
						So(conf, ShouldEqual, `{"isTLS":false}`)
						So(proto, ShouldEqual, "r66")
					}
				}

				So(rows.Err(), ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have restored the R66 agents", func() {
					rows, err := eng.DB.Query(`
					SELECT id,protocol,proto_config FROM local_agents UNION ALL 
					SELECT id,protocol,proto_config FROM remote_agents`)
					So(err, ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							id          int64
							proto, conf string
						)

						So(rows.Scan(&id, &proto, &conf), ShouldBeNil)
						So(id, ShouldBeBetweenOrEqual, 1, 4)
						So(proto, ShouldEqual, "r66")

						if id == 1 || id == 3 {
							So(conf, ShouldEqual, `{"isTLS":true}`)
						} else {
							So(conf, ShouldEqual, `{"isTLS":false}`)
						}
					}

					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}
