package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_5_2FillRemoteTransferID(eng *testEngine) {
	Convey("Given the 0.5.2 remote transfer id change", func() {
		setupDatabaseUpTo(eng, ver0_5_2FillRemoteTransferID{})

		_, err := eng.DB.Exec(`INSERT INTO transfers (id,owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,local_path,remote_path,filesize,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			(1234,'', '', false,1,0,0,'/loc_path1','/rem_path1',1111,'2000-01-02 04:05:06 -7:00',
			 '','',0,0,'','')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers (id,owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,local_path,remote_path,filesize,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			(5678,'', 'ABCD', true,1,0,0,'/loc_path2','/rem_path2',2222,'2000-01-03 04:05:06 -7:00',
			 '','',0,0,'','')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_5_2FillRemoteTransferID{})
			So(err, ShouldBeNil)

			Convey("Then it should have filled the remote transfer id", func() {
				rows, err := eng.DB.Query(`SELECT id,remote_transfer_id FROM transfers`)
				So(err, ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						id       int64
						remoteID string
					)
					So(rows.Scan(&id, &remoteID), ShouldBeNil)

					So(id, ShouldBeIn, []int64{1234, 5678})
					if id == 1234 {
						So(remoteID, ShouldEqual, "1234")
					} else {
						So(remoteID, ShouldEqual, "ABCD")
					}
				}

				So(rows.Err(), ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				err := eng.Downgrade(ver0_5_2FillRemoteTransferID{})
				So(err, ShouldBeNil)

				Convey("Then it should have re-emptied the remote transfer id", func() {
					rows, err := eng.DB.Query(`SELECT id,remote_transfer_id FROM transfers`)
					So(err, ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							id       int64
							remoteID string
						)
						So(rows.Scan(&id, &remoteID), ShouldBeNil)

						So(id, ShouldBeIn, []int64{1234, 5678})
						if id == 1234 {
							So(remoteID, ShouldEqual, "")
						} else {
							So(remoteID, ShouldEqual, "ABCD")
						}
					}

					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}
