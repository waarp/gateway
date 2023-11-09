package migrations

import (
	"runtime"

	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_9_0AddCloudInstances(eng *testEngine, dialect string) {
	Convey("Given the 0.9.0 cloud instances addition", func() {
		mig := ver0_9_0AddCloudInstances{}
		setupDatabaseUpTo(eng, mig)

		So(doesTableExist(eng.DB, dialect, "cloud_instances"), ShouldBeFalse)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added the new table", func() {
				So(doesTableExist(eng.DB, dialect, "cloud_instances"), ShouldBeTrue)
				tableShouldHaveColumns(eng.DB, "cloud_instances", "id", "owner",
					"name", "type", "key", "secret", "options")
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the new table", func() {
					So(doesTableExist(eng.DB, dialect, "cloud_instances"), ShouldBeFalse)
				})
			})
		})
	})
}

func testVer0_9_0LocalPathToURL(eng *testEngine) {
	osPath1, osPath2 := "/foo/bar/file.1", "/foo/bar/file.2"
	expURL1, expURL2 := "file:/foo/bar/file.1", "file:/foo/bar/file.2"

	if runtime.GOOS == "windows" {
		osPath1, osPath2 = `C:\foo\bar\file.1`, `C:\foo\bar\file.2`
		expURL1, expURL2 = `file:/C:/foo/bar/file.1`, `file:/C:/foo/bar/file.2`
	}

	Convey("Given the 0.9.0 local path to URL migration", func() {
		mig := ver0_9_0LocalPathToURL{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO rules 
    	(id, name, path, is_send) VALUES (1, 'recv', '/recv', false)`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_agents
    	(id, owner, name, protocol, address)
    	VALUES (10, 'gw', 'serv', 'sftp', 'localhost:10')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_accounts
    	(id, local_agent_id, login) VALUES (100, 10, 'user')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers
    	(id, owner, remote_transfer_id, rule_id, local_account_id, src_filename, local_path)
		VALUES (1000, 'gw', '1000', 1, 100, 'file.1', ?),
		       (2000, 'gw', '2000', 1, 100, 'file.2', ?)`,
			osPath1, osPath2)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have converted the local paths to URLs", func() {
				rows, queryErr := eng.DB.Query(`SELECT id, local_path FROM transfers
					ORDER BY id`)
				So(queryErr, ShouldBeNil)

				defer rows.Close()

				var (
					id   int64
					path string
				)

				So(rows.Err(), ShouldBeNil)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &path), ShouldBeNil)
				So(id, ShouldEqual, 1000)
				So(path, ShouldEqual, expURL1)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &path), ShouldBeNil)
				So(id, ShouldEqual, 2000)
				So(path, ShouldEqual, expURL2)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have converted the URLs back to local paths", func() {
					rows, queryErr := eng.DB.Query(`SELECT id, local_path 
						FROM transfers ORDER BY id`)
					So(queryErr, ShouldBeNil)

					defer rows.Close()

					var (
						id   int64
						path string
					)

					So(rows.Err(), ShouldBeNil)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id, &path), ShouldBeNil)
					So(id, ShouldEqual, 1000)
					So(path, ShouldEqual, osPath1)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id, &path), ShouldBeNil)
					So(id, ShouldEqual, 2000)
					So(path, ShouldEqual, osPath2)
				})
			})
		})
	})
}
