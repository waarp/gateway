package migrations

import . "github.com/smartystreets/goconvey/convey"

func testVer0_8_0DropNormalizedTransfersView(eng *testEngine) {
	Convey("Given the 0.8.0 normalized transfer view deletion", func() {
		mig := ver0_8_0DropNormalizedTransfersView{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have dropped the view", func() {
				_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
				So(err, ShouldNotBeNil)
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have restored the view", func() {
					_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func testVer0_8_0AddTransferFilename(eng *testEngine) {
	Convey("Given the 0.8.0 transfer filename addition", func() {
		mig := ver0_8_0AddTransferFilename{}
		setupDatabaseUpTo(eng, mig)

		// ### Rule ###
		_, err := eng.DB.Exec(`INSERT INTO rules(id, name, path, is_send) 
			VALUES (1, 'send', '/send', true), (2, 'recv', '/recv', false)`)
		So(err, ShouldBeNil)

		// ### Remote ###
		_, err = eng.DB.Exec(`INSERT INTO remote_agents(id, name, protocol, address) 
			VALUES (10, 'sftp_part', 'sftp', '1.1.1.1:1111')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id, remote_agent_id, login)
			VALUES (100, 10, 'toto')`)
		So(err, ShouldBeNil)

		// ### Local ###
		_, err = eng.DB.Exec(`INSERT INTO local_agents(id, owner, name, protocol, address) 
			VALUES (20, 'waarp_gw', 'sftp_serv', 'sftp', 'localhost:2222')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id, local_agent_id, login)
			VALUES (200, 20, 'tata')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id, owner, remote_transfer_id,
            rule_id, local_account_id, remote_account_id, local_path, remote_path)
            VALUES (1000, 'waarp_gw', 'push', 1, null, 100, '/loc/path/1', '/rem/path/1'),
                   (2000, 'waarp_gw', 'pull', 2, null, 100, '/loc/path/2', '/rem/path/2'),
                   (3000, 'waarp_gw', 'push', 2, 200, null, '/loc/path/3', '/rem/path/3'),
                   (4000, 'waarp_gw', 'pull', 1, 200, null, '/loc/path/4', '/rem/path/4')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added and filled the new column", func() {
				tableShouldHaveColumns(eng.DB, "transfers", "src_filename", "dest_filename")

				type row struct {
					ID                                  int64
					SrcFile, DestFile, LocPath, RemPath string
				}
				var rows []row

				queryAndParse(eng.DB, &rows, `SELECT id, src_filename, dest_filename,
       				local_path, remote_path	FROM transfers ORDER BY id`)

				So(rows, ShouldHaveLength, 4)

				So(rows[0], ShouldResemble, row{ // Client push
					ID:      1000,
					SrcFile: "/loc/path/1", DestFile: "/rem/path/1",
					LocPath: "/loc/path/1", RemPath: "/rem/path/1",
				})
				So(rows[1], ShouldResemble, row{ // Client pull
					ID:      2000,
					SrcFile: "/rem/path/2", DestFile: "/loc/path/2",
					LocPath: "/loc/path/2", RemPath: "/rem/path/2",
				})
				So(rows[2], ShouldResemble, row{ // Server push
					ID:      3000,
					SrcFile: "", DestFile: "/loc/path/3",
					LocPath: "/loc/path/3", RemPath: "",
				})
				So(rows[3], ShouldResemble, row{ // Server pull
					ID:      4000,
					SrcFile: "/loc/path/4", DestFile: "",
					LocPath: "/loc/path/4", RemPath: "",
				})
			})

			Convey("Then the local/remote path columns should no longer be mandatory", func() {
				_, err := eng.DB.Exec(`INSERT INTO transfers(id, owner, 
                    remote_transfer_id, rule_id, remote_account_id, src_filename)
            		VALUES (5000, 'waarp_gw', 'new', 1, 100, 'file5')`)
				So(err, ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the filename column", func() {
					tableShouldNotHaveColumns(eng.DB, "transfers", "src_filename",
						"dest_filename")
				})

				Convey("Then the local/remote path columns should again be mandatory", func() {
					_, err := eng.DB.Exec(`INSERT INTO transfers(id, owner, 
                    	remote_transfer_id, rule_id, remote_account_id)
            		VALUES (3000, 'waarp_gw', 'new', 1, 100)`)
					So(err, ShouldNotBeNil)
				})

				Convey("Then it should have restored the remote path column for server transfers", func() {
					rows, err := eng.DB.Query(`SELECT remote_path FROM transfers
					WHERE local_account_id IS NOT NULL`)
					So(err, ShouldBeNil)
					defer rows.Close()

					for rows.Next() {
						var remPath string
						So(rows.Scan(&remPath), ShouldBeNil)
						So(remPath, ShouldNotBeBlank)
					}

					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}

func testVer0_8_0AddHistoryFilename(eng *testEngine) {
	Convey("Given the 0.8.0 history filename addition", func() {
		mig := ver0_8_0AddHistoryFilename{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,remote_transfer_id,
			is_server,is_send,account,agent,protocol,remote_path,local_path,rule,
			start,status,step)
            VALUES (1,'waarp_gw','111',false,true,'acc','ag','proto','/rem/path/1',
                    '/loc/path/1','push','2021-01-01 02:00:00.123456','DONE','StepNone'),
                   (2,'waarp_gw','222',false,false,'acc','ag','proto','/rem/path/2',
                    '/loc/path/2','pull','2021-01-01 02:00:00.123456','DONE','StepNone'),
                   (3,'waarp_gw','333',true,false,'acc','ag','proto','/rem/path/3',
                    '/loc/path/3','push','2021-01-01 02:00:00.123456','DONE','StepNone'),
                   (4,'waarp_gw','444',true,true,'acc','ag','proto','/rem/path/4',
                    '/loc/path/4','pull','2021-01-01 02:00:00.123456','DONE','StepNone')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added and filled the new column", func() {
				tableShouldHaveColumns(eng.DB, "transfer_history", "src_filename",
					"dest_filename")

				type row struct {
					ID                                  int64
					SrcFile, DestFile, LocPath, RemPath string
				}
				var rows []row

				queryAndParse(eng.DB, &rows, `SELECT id, src_filename, dest_filename,
       				local_path, remote_path	FROM transfer_history ORDER BY id`)

				So(rows, ShouldHaveLength, 4)

				So(rows[0], ShouldResemble, row{ // Client push
					ID:      1,
					SrcFile: "/loc/path/1", DestFile: "/rem/path/1",
					LocPath: "/loc/path/1", RemPath: "/rem/path/1",
				})
				So(rows[1], ShouldResemble, row{ // Client pull
					ID:      2,
					SrcFile: "/rem/path/2", DestFile: "/loc/path/2",
					LocPath: "/loc/path/2", RemPath: "/rem/path/2",
				})
				So(rows[2], ShouldResemble, row{ // Server push
					ID:      3,
					SrcFile: "", DestFile: "/loc/path/3",
					LocPath: "/loc/path/3", RemPath: "",
				})
				So(rows[3], ShouldResemble, row{ // Server pull
					ID:      4,
					SrcFile: "/loc/path/4", DestFile: "",
					LocPath: "/loc/path/4", RemPath: "",
				})
			})

			Convey("Then the local/remote path columns should no longer be mandatory", func() {
				_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,
					remote_transfer_id,is_server,is_send,account,agent,protocol,
                    src_filename,rule,start,status,step)
            		VALUES (5,'waarp_gw','555',false,true,'acc','ag','proto','file5',
							'push','2021-01-01 02:00:00.123456','DONE','StepNone')`)
				So(err, ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the filename column", func() {
					tableShouldNotHaveColumns(eng.DB, "transfer_history",
						"src_filename", "dest_filename")
				})

				Convey("Then the local/remote path columns should again be mandatory", func() {
					_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,
					remote_transfer_id,is_server,is_send,account,agent,protocol,
                    rule,start,status,step)
            		VALUES (5,'waarp_gw','555',false,true,'acc','ag','proto','push',
            		        '2021-01-01 02:00:00.123456','DONE','StepNone')`)
					So(err, ShouldNotBeNil)
				})

				Convey("Then it should have restored the remote path column for server transfers", func() {
					rows, err := eng.DB.Query(`SELECT remote_path 
						FROM transfer_history WHERE is_server=true`)
					So(err, ShouldBeNil)
					defer rows.Close()

					for rows.Next() {
						var remPath string
						So(rows.Scan(&remPath), ShouldBeNil)
						So(remPath, ShouldNotBeBlank)
					}

					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}

func testVer0_8_0UpdateNormalizedTransfersView(eng *testEngine) {
	Convey("Given the 0.8.0 normalized transfer view restoration", func() {
		mig := ver0_8_0UpdateNormalizedTransfersView{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
		So(err, ShouldNotBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have restored the view", func() {
				_, err := eng.DB.Exec(`SELECT src_filename, dest_filename
					FROM normalized_transfers`)
				So(err, ShouldBeNil)
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have drop the view", func() {
					_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}
