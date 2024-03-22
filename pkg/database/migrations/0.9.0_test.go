package migrations

import (
	"database/sql"

	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_9_0FixLocalServerEnabled(eng *testEngine, dialect string) {
	Convey("Given the 0.9.0 local agent 'enable' column replacement", func() {
		mig := ver0_9_0FixLocalServerEnabled{}
		setupDatabaseUpTo(eng, mig)

		tableShouldHaveColumns(eng.DB, "local_agents", "enabled")
		tableShouldNotHaveColumns(eng.DB, "local_agents", "disabled")

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have renamed the column", func() {
				tableShouldNotHaveColumns(eng.DB, "local_agents", "enabled")
				tableShouldHaveColumns(eng.DB, "local_agents", "disabled")

				Convey(`Then the "normalized_transfer" view should still
					exist (sqlite only)`, func() {
					if dialect == SQLite {
						_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
						So(err, ShouldBeNil)
					}
				})
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have restored the column", func() {
					tableShouldHaveColumns(eng.DB, "local_agents", "enabled")
					tableShouldNotHaveColumns(eng.DB, "local_agents", "disabled")

					Convey(`Then the "normalized_transfer" view should still
						exist (sqlite only)`, func() {
						if dialect == SQLite {
							_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
							So(err, ShouldBeNil)
						}
					})
				})
			})
		})
	})
}

func testVer0_9_0AddClientsTable(eng *testEngine, dialect string) {
	Convey("Given the 0.9.0 'clients' table creation", func() {
		mig := ver0_9_0AddClientsTable{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO users(owner,username)
			VALUES ('bar','user_b1'), ('foo','user_f1'), ('bar','user_b2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
    		remote_agents(name,protocol,address)
    		VALUES ('p1','proto1','1.1.1.1'),
    		       ('p2','proto2','2.2.2.2'),
    		       ('p3','proto3','3.3.3.3')`)
		So(err, ShouldBeNil)

		So(doesTableExist(eng.DB, dialect, "clients"), ShouldBeFalse)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added the table", func() {
				So(doesTableExist(eng.DB, dialect, "clients"), ShouldBeTrue)
				tableShouldHaveColumns(eng.DB, "clients", "id", "owner", "name",
					"protocol", "disabled", "local_address", "proto_config")

				Convey("Then it should have inserted 1 client per protocol", func() {
					rows, err := eng.DB.Query(`SELECT id,owner,name,protocol,
       				local_address,proto_config FROM clients ORDER BY id`)
					So(err, ShouldBeNil)

					defer rows.Close()

					type client struct{ owner, proto, name string }

					expected := map[int64]client{
						1: {"bar", "proto1", "proto1"},
						2: {"foo", "proto1", "proto1"},
						3: {"bar", "proto2", "proto2"},
						4: {"foo", "proto2", "proto2"},
						5: {"bar", "proto3", "proto3"},
						6: {"foo", "proto3", "proto3"},
					}

					actual := map[int64]client{}

					for rows.Next() {
						var (
							id                               int64
							owner, name, proto, addr, config string
						)

						So(rows.Scan(&id, &owner, &name, &proto, &addr, &config), ShouldBeNil)
						So(addr, ShouldBeBlank)
						So(config, ShouldEqual, "{}")

						actual[id] = client{owner: owner, proto: proto, name: name}
					}

					So(rows.Err(), ShouldBeNil)
					So(actual, ShouldResemble, expected)
				})
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the table", func() {
					So(doesTableExist(eng.DB, dialect, "clients"), ShouldBeFalse)
				})
			})
		})
	})
}

func testVer0_9_0AddRemoteAgentOwner(eng *testEngine) {
	Convey("Given the 0.9.0 'owner' column addition", func() {
		mig := ver0_9_0AddRemoteAgentOwner{}
		setupDatabaseUpTo(eng, mig)

		tableShouldNotHaveColumns(eng.DB, "remote_agents", "owner")

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added the column", func() {
				tableShouldHaveColumns(eng.DB, "remote_agents", "owner")
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the column", func() {
					tableShouldNotHaveColumns(eng.DB, "remote_agents", "owner")
				})
			})
		})
	})
}

//nolint:maintidx //function is complex because we must check lots of parameters
func testVer0_9_0DuplicateRemoteAgents(eng *testEngine, dialect string) {
	Convey("Given the 0.9.0 partner duplication", func() {
		mig := ver0_9_0DuplicateRemoteAgents{currentOwner: "aaa"}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO users(owner,username)
			VALUES ('aaa','user_a1'), ('bbb','user_b1'), ('bbb','user_b2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
    		remote_agents(id,name,protocol,proto_config,address)
    		VALUES (1,'proto1_part','proto1','{}','addr1'),
    		       (2,'proto2_part','proto2','{}','addr2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
    		remote_accounts(id,remote_agent_id,login,password)
    		VALUES (11,1,'proto1_acc','sesame1'),
    		       (12,2,'proto2_acc','sesame2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rules(id,name,is_send,path)
			VALUES (10000,'push',true,'/push'), (20000,'pull',false,'/pull')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rule_access(remote_agent_id,remote_account_id,rule_id)
			VALUES (1,null,10000), (null,12,20000)`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
			crypto_credentials(id,remote_agent_id,remote_account_id,name,
			                   private_key,ssh_public_key,certificate)
			VALUES (101,  1 ,null,'proto1_part_crypto','pk1','pbk1','cert1'),
			       (102,null, 12 ,'proto2_acc_crypto' ,'pk2','pbk2','cert2')`)
		So(err, ShouldBeNil)

		// Pgsql does not increment the sequence when IDs are inserted manually,
		// so we have to manually increment the sequences to keep the test
		// consistent with other databases.
		if dialect == PostgreSQL {
			_, err = eng.DB.Exec(`SELECT setval('remote_agents_id_seq', 2);
				SELECT setval('remote_accounts_id_seq', 12);
				SELECT setval('crypto_credentials_id_seq', 102)`)
			So(err, ShouldBeNil)
		}

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have duplicated the partners", func() {
				rows, err := eng.DB.Query(`SELECT id,name,owner,proto_config,
			       	address FROM remote_agents ORDER BY id`)
				So(err, ShouldBeNil)
				Reset(func() { So(rows.Close(), ShouldBeNil) })

				type partner struct {
					name, owner, config, addr string
				}

				expected := map[int64]partner{
					1: {"aaa", "proto1_part", "{}", "addr1"},
					2: {"aaa", "proto2_part", "{}", "addr2"},
					3: {"bbb", "proto1_part", "{}", "addr1"},
					4: {"bbb", "proto2_part", "{}", "addr2"},
				}

				actual := map[int64]partner{}

				for rows.Next() {
					var (
						id                        int64
						name, owner, config, addr string
					)

					So(rows.Scan(&id, &name, &owner, &config, &addr), ShouldBeNil)
					So(actual, ShouldNotContainKey, id)

					actual[id] = partner{owner, name, config, addr}
				}

				So(rows.Err(), ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("Then it should have duplicated the accounts", func() {
				rows, err := eng.DB.Query(`SELECT id,remote_agent_id,login,password
       				FROM remote_accounts ORDER BY id`)
				So(err, ShouldBeNil)
				Reset(func() { So(rows.Close(), ShouldBeNil) })

				type account struct {
					partnerID       int64
					login, password string
				}

				expected := map[int64]account{
					11: {1, "proto1_acc", "sesame1"},
					12: {2, "proto2_acc", "sesame2"},
					13: {3, "proto1_acc", "sesame1"},
					14: {4, "proto2_acc", "sesame2"},
				}

				actual := map[int64]account{}

				for rows.Next() {
					var (
						id, partnerID   int64
						login, password string
					)

					So(rows.Scan(&id, &partnerID, &login, &password), ShouldBeNil)
					So(actual, ShouldNotContainKey, id)

					actual[id] = account{partnerID, login, password}
				}

				So(rows.Err(), ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("Then it should have duplicated the crypto credentials", func() {
				rows, err := eng.DB.Query(`SELECT id,remote_agent_id,remote_account_id,
       				name,private_key,ssh_public_key,certificate FROM crypto_credentials
       				ORDER BY id`)
				So(err, ShouldBeNil)
				Reset(func() { So(rows.Close(), ShouldBeNil) })

				type crypto struct {
					remAgID, remAccID   int64
					name, pk, pbk, cert string
				}

				expected := map[int64]crypto{
					101: {1, 0, "proto1_part_crypto", "pk1", "pbk1", "cert1"},
					102: {0, 12, "proto2_acc_crypto", "pk2", "pbk2", "cert2"},
					103: {3, 0, "proto1_part_crypto", "pk1", "pbk1", "cert1"},
					104: {0, 14, "proto2_acc_crypto", "pk2", "pbk2", "cert2"},
				}

				actual := map[int64]crypto{}

				for rows.Next() {
					var (
						id                  int64
						remAgID, remAccID   sql.NullInt64
						name, pk, pbk, cert string
					)

					So(rows.Scan(&id, &remAgID, &remAccID, &name, &pk, &pbk, &cert), ShouldBeNil)
					So(actual, ShouldNotContainKey, id)

					actual[id] = crypto{remAgID.Int64, remAccID.Int64, name, pk, pbk, cert}
				}

				So(rows.Err(), ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("Then it should have duplicated the rule accesses", func() {
				rows, err := eng.DB.Query(`SELECT remote_agent_id,remote_account_id,
       				rule_id FROM rule_access ORDER BY rule_id,remote_account_id,remote_agent_id`)
				So(err, ShouldBeNil)
				Reset(func() { So(rows.Close(), ShouldBeNil) })

				type ruleAccess struct{ remAgID, remAccID, ruleID int64 }

				expected := []ruleAccess{
					{1, 0, 10000},
					{3, 0, 10000},
					{0, 12, 20000},
					{0, 14, 20000},
				}

				var actual []ruleAccess

				for rows.Next() {
					var (
						remAgID, remAccID sql.NullInt64
						ruleID            int64
					)

					So(rows.Scan(&remAgID, &remAccID, &ruleID), ShouldBeNil)

					actual = append(actual, ruleAccess{remAgID.Int64, remAccID.Int64, ruleID})
				}

				So(rows.Err(), ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have removed the duplicated partners", func() {
					var id int64

					rows, err := eng.DB.Query(`SELECT id FROM remote_agents
          				ORDER BY id`)
					So(err, ShouldBeNil)
					Reset(func() { So(rows.Close(), ShouldBeNil) })

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 1)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 2)

					So(rows.Next(), ShouldBeFalse)
					So(rows.Err(), ShouldBeNil)
				})

				Convey("Then it should have removed the duplicated accounts", func() {
					var id int64

					rows, err := eng.DB.Query(`SELECT id FROM remote_accounts
                    	ORDER BY id`)
					So(err, ShouldBeNil)
					Reset(func() { So(rows.Close(), ShouldBeNil) })

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 11)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 12)

					So(rows.Next(), ShouldBeFalse)
					So(rows.Err(), ShouldBeNil)
				})

				Convey("Then it should have removed the duplicated cryptos", func() {
					var id int64

					rows, err := eng.DB.Query(`SELECT id FROM crypto_credentials
          				ORDER BY id`)
					So(err, ShouldBeNil)
					Reset(func() { So(rows.Close(), ShouldBeNil) })

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 101)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&id), ShouldBeNil)
					So(id, ShouldEqual, 102)

					So(rows.Next(), ShouldBeFalse)
					So(rows.Err(), ShouldBeNil)
				})

				Convey("Then it should have removed the duplicated rule accesses", func() {
					var remAgID, remAccID sql.NullInt64

					rows, err := eng.DB.Query(`SELECT remote_agent_id,remote_account_id
						FROM rule_access ORDER BY rule_id,remote_account_id,remote_agent_id`)
					So(err, ShouldBeNil)
					Reset(func() { So(rows.Close(), ShouldBeNil) })

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&remAgID, &remAccID), ShouldBeNil)
					So(remAgID.Int64, ShouldEqual, 1)

					So(rows.Next(), ShouldBeTrue)
					So(rows.Scan(&remAgID, &remAccID), ShouldBeNil)
					So(remAccID.Int64, ShouldEqual, 12)

					So(rows.Next(), ShouldBeFalse)
					So(rows.Err(), ShouldBeNil)
				})
			})
		})
	})
}

func testVer0_9_0RelinkTransfers(eng *testEngine) {
	Convey("Given the 0.9.0 transfer agent relink", func() {
		mig := ver0_9_0RelinkTransfers{currentOwner: "aaa"}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO
    		remote_agents(id,owner,name,protocol,address)
    		VALUES (10, 'aaa','proto1_partner', 'proto1', '1.1.1.1'),
    		       (20, 'aaa','proto2_partner', 'proto2', '2.2.2.2'),
    		       (30, 'bbb','proto1_partner', 'proto1', '1.1.1.1'),
    		       (40, 'bbb','proto2_partner', 'proto2', '2.2.2.2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
    		remote_accounts(id,remote_agent_id,login)
    		VALUES (100, 10, 'proto1_account'), (200, 20, 'proto2_account'),
    		       (300, 30, 'proto1_account'), (400, 40, 'proto2_account')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rules(id,name,is_send,path)
			VALUES (1000,'push',true,'/push')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
                rule_id,remote_account_id,local_path,remote_path)
            VALUES (10000, 'aaa', 'proto1a', 1000, 100, '/loc/path', '/rem/path'),
                   (20000, 'aaa', 'proto2a', 1000, 200, '/loc/path', '/rem/path'),
                   (30000, 'bbb', 'proto1b', 1000, 100, '/loc/path', '/rem/path'),
                   (40000, 'bbb', 'proto2b', 1000, 200, '/loc/path', '/rem/path')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have updated the relevant transfers", func() {
				rows, err := eng.DB.Query(`SELECT id,owner,remote_account_id
					FROM transfers ORDER BY id`)
				So(err, ShouldBeNil)
				Reset(func() { So(rows.Close(), ShouldBeNil) })

				type trans struct {
					owner    string
					remAccID int64
				}

				expected := map[int64]trans{
					10000: {"aaa", 100}, 20000: {"aaa", 200},
					30000: {"bbb", 300}, 40000: {"bbb", 400},
				}

				actual := map[int64]trans{}

				for rows.Next() {
					var (
						id, remAccID int64
						owner        string
					)

					So(rows.Scan(&id, &owner, &remAccID), ShouldBeNil)

					actual[id] = trans{owner: owner, remAccID: remAccID}
				}

				So(rows.Err(), ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have reverted the changes", func() {
					rows, err := eng.DB.Query(`SELECT id,owner,remote_account_id
					FROM transfers ORDER BY id`)
					So(err, ShouldBeNil)
					Reset(func() { So(rows.Close(), ShouldBeNil) })

					type trans struct {
						owner    string
						remAccID int64
					}

					expected := map[int64]trans{
						10000: {"aaa", 100}, 20000: {"aaa", 200},
						30000: {"bbb", 100}, 40000: {"bbb", 200},
					}

					actual := map[int64]trans{}

					for rows.Next() {
						var (
							id, remAccID int64
							owner        string
						)

						So(rows.Scan(&id, &owner, &remAccID), ShouldBeNil)

						actual[id] = trans{owner: owner, remAccID: remAccID}
					}

					So(rows.Err(), ShouldBeNil)
					So(actual, ShouldResemble, expected)
				})
			})
		})
	})
}

func testVer0_9_0AddTransfersClientID(eng *testEngine) {
	Convey("Given the 0.9.0 transfer client_id addition", func() {
		mig := ver0_9_0AddTransferClientID{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO clients (id, name, owner, protocol)
			VALUES (1, 'proto1', 'aaa', 'proto1'), (2, 'proto2', 'aaa', 'proto2'),
			       (3, 'proto1', 'bbb', 'proto1'), (4, 'proto2', 'bbb', 'proto2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(
			`INSERT INTO remote_agents(id, name, owner, protocol, address) 
			VALUES (10, 'partner1', 'aaa', 'proto1', 'addr1'),
			       (20, 'partner2', 'aaa', 'proto2', 'addr2'),
			       (30, 'partner1', 'bbb', 'proto1', 'addr1'),
			       (40, 'partner2', 'bbb', 'proto2', 'addr2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO
    		remote_accounts(id, remote_agent_id, login)
    		VALUES (100, 10, 'account1'), (200, 20, 'account2'),
    		       (300, 30, 'account1'), (400, 40, 'account2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rules(id, name, is_send, path)
			VALUES (1000, 'push', true, '/push')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id, owner, remote_transfer_id,
                rule_id, remote_account_id, local_path, remote_path)
            VALUES (10000, 'aaa', 'proto1a', 1000, 100, '/loc/path', '/rem/path'),
                   (20000, 'aaa', 'proto2a', 1000, 200, '/loc/path', '/rem/path'),
                   (30000, 'bbb', 'proto1b', 1000, 300, '/loc/path', '/rem/path'),
                   (40000, 'bbb', 'proto2b', 1000, 400, '/loc/path', '/rem/path')`)
		So(err, ShouldBeNil)

		tableShouldNotHaveColumns(eng.DB, "transfers", "client_id")

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added and filled the 'client_id' column", func() {
				tableShouldHaveColumns(eng.DB, "transfers", "client_id")

				rows, queryErr := eng.DB.Query(`SELECT id, client_id 
					FROM transfers ORDER BY id`)
				So(queryErr, ShouldBeNil)

				defer rows.Close()

				var id, clientID int64

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &clientID), ShouldBeNil)
				So(id, ShouldEqual, 10000)
				So(clientID, ShouldEqual, 1)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &clientID), ShouldBeNil)
				So(id, ShouldEqual, 20000)
				So(clientID, ShouldEqual, 2)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &clientID), ShouldBeNil)
				So(id, ShouldEqual, 30000)
				So(clientID, ShouldEqual, 3)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &clientID), ShouldBeNil)
				So(id, ShouldEqual, 40000)
				So(clientID, ShouldEqual, 4)

				So(rows.Next(), ShouldBeFalse)
				So(rows.Err(), ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the 'client_id' column", func() {
					tableShouldNotHaveColumns(eng.DB, "transfers", "client_id")
				})
			})
		})
	})
}

func testVer0_9_0AddHistoryClient(eng *testEngine) {
	Convey("Given the 0.9.0 history client addition", func() {
		mig := ver0_9_0AddHistoryClient{}
		setupDatabaseUpTo(eng, mig)

		_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,
        	remote_transfer_id,is_server,is_send,rule,account,agent,
            protocol,local_path,remote_path,start,stop,status,step) VALUES
            (1,'wg','abc',true,true,'push','loc_ag','loc_acc','proto1','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone'),
        	(2,'wg','def',false,true,'push','rem_ag1','rem_acc1','proto1','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone'),
            (3,'wg','ghi',false,false,'pull','rem_ag2','rem_acc2','proto2','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added and filled the 'client' column", func() {
				tableShouldHaveColumns(eng.DB, "transfer_history", "client")

				rows, queryErr := eng.DB.Query(`SELECT id, client FROM transfer_history
					ORDER BY id`)
				So(queryErr, ShouldBeNil)

				defer rows.Close()

				var (
					id     int64
					client string
				)

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &client), ShouldBeNil)
				So(id, ShouldEqual, 1)
				So(client, ShouldEqual, "")

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &client), ShouldBeNil)
				So(id, ShouldEqual, 2)
				So(client, ShouldEqual, "proto1_client")

				So(rows.Next(), ShouldBeTrue)
				So(rows.Scan(&id, &client), ShouldBeNil)
				So(id, ShouldEqual, 3)
				So(client, ShouldEqual, "proto2_client")

				So(rows.Next(), ShouldBeFalse)
				So(rows.Err(), ShouldBeNil)
			})

			Convey("When reverting the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the 'client' column", func() {
					tableShouldNotHaveColumns(eng.DB, "transfer_history", "client")
				})
			})
		})
	})
}

func testVer0_9_0AddNormalizedTransfersView(eng *testEngine) {
	Convey("Given the 0.9.0 normalized transfer view restoration", func() {
		mig := ver0_9_0RestoreNormalizedTransfersView{}
		setupDatabaseUpTo(eng, mig)

		// ### CLIENTS ###
		_, err := eng.DB.Exec(`INSERT INTO clients(id, owner, name, protocol) 
			VALUES (2222, 'bbb', 'sftp', 'sftp')`)
		So(err, ShouldBeNil)

		// ### RULES ###
		_, err = eng.DB.Exec(`INSERT INTO rules(id, name, is_send, path) 
			VALUES (1, 'push', TRUE, '/push'), (2, 'pull', FALSE, '/pull')`)
		So(err, ShouldBeNil)

		// ### LOCAL ###
		_, err = eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,address)
			VALUES (10, 'aaa', 'sftp_serv', 'sftp', '1.1.1.1:1111')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100, 10, 'toto')`)
		So(err, ShouldBeNil)

		// ### REMOTE ###
		_, err = eng.DB.Exec(`INSERT INTO remote_agents(id,owner,name,protocol,address) 
			VALUES (20, 'bbb', 'sftp_part', 'sftp', '2.2.2.2:2222')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200, 20, 'tata')`)
		So(err, ShouldBeNil)

		// ### TRANSFERS ###
		_, err = eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,rule_id,
            client_id,local_account_id,remote_account_id,src_filename,dest_filename)
            VALUES(1000, 'aaa', 'abcd', 1, NULL, 100, NULL, '/src/1', '/dst/1'),
                  (2000, 'bbb', 'efgh', 2, 2222, NULL, 200, '/src/2', '/dst/2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            	is_send,remote_transfer_id,rule,client,account,agent,protocol,
                src_filename,dest_filename,filesize,start,stop,status,step) 
			VALUES (3000,'ccc',FALSE,TRUE,'xyz','push','r66_client','tutu',
				'r66_part','r66','/src/3','/dst/3',123,'2021-01-03 01:00:00',
			    '2021-01-03 02:00:00','CANCELLED','StepData')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			So(eng.Upgrade(mig), ShouldBeNil)

			Convey("Then it should have added the view", func() {
				type normTrans struct {
					id                                  int64
					isServ, isSend, isTrans             bool
					rule, client, account, agent, proto string
					srcFile, dstFile                    string
				}

				rows, err := eng.DB.Query(`SELECT id,is_server,is_send,is_transfer,
       				rule,client,account,agent,protocol,src_filename,dest_filename
					FROM normalized_transfers ORDER BY id`)
				So(err, ShouldBeNil)
				defer rows.Close()

				transfers := make([]normTrans, 0, 3)

				for rows.Next() {
					var trans normTrans

					So(rows.Scan(&trans.id, &trans.isServ, &trans.isSend,
						&trans.isTrans, &trans.rule, &trans.client, &trans.account,
						&trans.agent, &trans.proto, &trans.srcFile, &trans.dstFile),
						ShouldBeNil)

					transfers = append(transfers, trans)
				}

				So(rows.Err(), ShouldBeNil)

				So(transfers, ShouldResemble, []normTrans{
					{
						id: 1000, isServ: true, isSend: true, isTrans: true,
						rule: "push", client: "", account: "toto",
						agent: "sftp_serv", proto: "sftp",
						srcFile: "/src/1", dstFile: "/dst/1",
					},
					{
						id: 2000, isServ: false, isSend: false, isTrans: true,
						rule: "pull", client: "sftp", account: "tata",
						agent: "sftp_part", proto: "sftp",
						srcFile: "/src/2", dstFile: "/dst/2",
					},
					{
						id: 3000, isServ: false, isSend: true, isTrans: false,
						rule: "push", client: "r66_client", account: "tutu",
						agent: "r66_part", proto: "r66",
						srcFile: "/src/3", dstFile: "/dst/3",
					},
				})
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(mig), ShouldBeNil)

				Convey("Then it should have dropped the view", func() {
					_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}
