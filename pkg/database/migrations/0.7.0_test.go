package migrations

import (
	"database/sql"
	"fmt"
	"time"

	"code.waarp.fr/lib/migration"
	. "github.com/smartystreets/goconvey/convey"
)

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

func testVer0_7_0RevampUsersTable(eng *testEngine, dialect string) {
	Convey("Given the 0.7.0 'users' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampUsersTable{})

		if dialect == migration.PostgreSQL {
			_, err := eng.DB.Exec(`INSERT INTO users(id,owner,username,
            	password_hash,permissions) VALUES (1,'waarp_gw','toto',
				'pswd_hash',$1)`, []byte{0x0A, 0x00, 0x00, 0x00})
			So(err, ShouldBeNil)
		} else {
			_, err := eng.DB.Exec(`INSERT INTO users(id,owner,username,
				password_hash,permissions) VALUES (1,'waarp_gw','toto',
				'pswd_hash',?)`, []byte{0x0A, 0x00, 0x00, 0x00})
			So(err, ShouldBeNil)
		}

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampUsersTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,owner,username,password_hash,
       				permissions FROM users`)
				So(err, ShouldBeNil)

				var id, perm int64
				var owner, name, hash string

				So(row.Scan(&id, &owner, &name, &hash, &perm), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(owner, ShouldEqual, "waarp_gw")
				So(name, ShouldEqual, "toto")
				So(hash, ShouldEqual, "pswd_hash")
				So(perm, ShouldEqual, 10)
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampUsersTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,owner,username,password_hash,
       				permissions FROM users`)
					So(err, ShouldBeNil)

					var id int64
					var owner, name, hash string
					var perm []byte

					So(row.Scan(&id, &owner, &name, &hash, &perm), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(owner, ShouldEqual, "waarp_gw")
					So(name, ShouldEqual, "toto")
					So(hash, ShouldEqual, "pswd_hash")
					So(perm, ShouldResemble, []byte{0x0A, 0x00, 0x00, 0x00})
				})
			})
		})
	})
}

func testVer0_7_0RevampLocalAgentTable(eng *testEngine) {
	Convey("Given the 0.7.0 'local_agents' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampLocalAgentsTable{})

		_, err := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (1,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampLocalAgentsTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,owner,name,protocol,address,root_dir,
       				receive_dir,send_dir,tmp_receive_dir,proto_config FROM local_agents`)
				So(err, ShouldBeNil)

				var id int64
				var owner, name, proto, addr, root, recv, send, tmp, conf string

				So(row.Scan(&id, &owner, &name, &proto, &addr, &root, &recv,
					&send, &tmp, &conf), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(owner, ShouldEqual, "waarp_gw")
				So(name, ShouldEqual, "sftp_serv")
				So(proto, ShouldEqual, "sftp")
				So(addr, ShouldEqual, "localhost:2222")
				So(root, ShouldEqual, "root")
				So(recv, ShouldEqual, "rcv")
				So(send, ShouldEqual, "snd")
				So(tmp, ShouldEqual, "tmp")
				So(conf, ShouldEqual, "{}")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampLocalAgentsTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,owner,name,protocol,address,root_dir,
       				receive_dir,send_dir,tmp_receive_dir,proto_config FROM local_agents`)
					So(err, ShouldBeNil)

					var id int64
					var owner, name, proto, addr, root, recv, send, tmp string
					var conf []byte

					So(row.Scan(&id, &owner, &name, &proto, &addr, &root, &recv,
						&send, &tmp, &conf), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(owner, ShouldEqual, "waarp_gw")
					So(name, ShouldEqual, "sftp_serv")
					So(proto, ShouldEqual, "sftp")
					So(addr, ShouldEqual, "localhost:2222")
					So(root, ShouldEqual, "root")
					So(recv, ShouldEqual, "rcv")
					So(send, ShouldEqual, "snd")
					So(tmp, ShouldEqual, "tmp")
					So(conf, ShouldResemble, []byte("{}"))
				})
			})
		})
	})
}

func testVer0_7_0RevampRemoteAgentTable(eng *testEngine) {
	Convey("Given the 0.7.0 'remote_agents' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampRemoteAgentsTable{})

		_, err := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
			proto_config) VALUES (1,'sftp_part','sftp','localhost:2222','{}')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampRemoteAgentsTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,name,protocol,address,
       				proto_config FROM remote_agents`)
				So(err, ShouldBeNil)

				var id int64
				var name, proto, addr, conf string

				So(row.Scan(&id, &name, &proto, &addr, &conf), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(name, ShouldEqual, "sftp_part")
				So(proto, ShouldEqual, "sftp")
				So(addr, ShouldEqual, "localhost:2222")
				So(conf, ShouldEqual, "{}")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampRemoteAgentsTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,name,protocol,address,
       					proto_config FROM remote_agents`)
					So(err, ShouldBeNil)

					var id int64
					var name, proto, addr string
					var conf []byte

					So(row.Scan(&id, &name, &proto, &addr, &conf), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(name, ShouldEqual, "sftp_part")
					So(proto, ShouldEqual, "sftp")
					So(addr, ShouldEqual, "localhost:2222")
					So(conf, ShouldResemble, []byte("{}"))
				})
			})
		})
	})
}

func testVer0_7_0RevampLocalAccountsTable(eng *testEngine, dialect string) {
	Convey("Given the 0.7.0 'local_accounts' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampLocalAccountsTable{})

		_, err := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (1,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		if dialect == migration.PostgreSQL {
			_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,
				login,password_hash) VALUES (1,1,'toto',$1)`, []byte("pswdhash"))
		} else {
			_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,
				login,password_hash) VALUES (1,1,'toto',?)`, []byte("pswdhash"))
		}

		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampLocalAccountsTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,local_agent_id,login,
       				password_hash FROM local_accounts`)
				So(err, ShouldBeNil)

				var id, agID int64
				var login, hash string

				So(row.Scan(&id, &agID, &login, &hash), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(agID, ShouldEqual, 1)
				So(login, ShouldEqual, "toto")
				So(hash, ShouldEqual, "pswdhash")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampLocalAccountsTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,local_agent_id,login,
       					password_hash FROM local_accounts`)
					So(err, ShouldBeNil)

					var id, agID int64
					var login, hash string

					So(row.Scan(&id, &agID, &login, &hash), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(agID, ShouldEqual, 1)
					So(login, ShouldEqual, "toto")
					So(hash, ShouldEqual, "pswdhash")
				})
			})
		})
	})
}

func testVer0_7_0RevampRemoteAccountsTable(eng *testEngine, dialect string) {
	Convey("Given the 0.7.0 'remote_accounts' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampRemoteAccountsTable{})

		_, err := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (1,'sftp_part','sftp','localhost:2222','{}')`)
		So(err, ShouldBeNil)

		if dialect == migration.PostgreSQL {
			_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,
				login,password) VALUES (1,1,'toto',$1)`, []byte("pswd"))
		} else {
			_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,
				login,password) VALUES (1,1,'toto',?)`, []byte("pswd"))
		}

		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampRemoteAccountsTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,remote_agent_id,login,
       				password FROM remote_accounts`)
				So(err, ShouldBeNil)

				var id, agID int64
				var login, pwd string

				So(row.Scan(&id, &agID, &login, &pwd), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(agID, ShouldEqual, 1)
				So(login, ShouldEqual, "toto")
				So(pwd, ShouldEqual, "pswd")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampRemoteAccountsTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,remote_agent_id,login,
       					password FROM remote_accounts`)
					So(err, ShouldBeNil)

					var id, agID int64
					var login, hash string

					So(row.Scan(&id, &agID, &login, &hash), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(agID, ShouldEqual, 1)
					So(login, ShouldEqual, "toto")
					So(hash, ShouldEqual, "pswd")
				})
			})
		})
	})
}

func testVer0_7_0RevampRulesTable(eng *testEngine) {
	Convey("Given the 0.7.0 'rules' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampRulesTable{})

		_, err := eng.DB.Exec(`INSERT INTO rules(id,name,send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampRulesTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,name,is_send,comment,path,
            		local_dir,remote_dir,tmp_local_receive_dir FROM rules`)
				So(err, ShouldBeNil)

				var id int64
				var name, comment, path, loc, rem, tmp string
				var send bool

				So(row.Scan(&id, &name, &send, &comment, &path, &loc, &rem,
					&tmp), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(name, ShouldEqual, "push")
				So(send, ShouldEqual, true)
				So(comment, ShouldEqual, "this is a comment")
				So(path, ShouldEqual, "/push")
				So(loc, ShouldEqual, "locDir")
				So(rem, ShouldEqual, "remDir")
				So(tmp, ShouldEqual, "tmpDir")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampRulesTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,name,send,comment,path,
            		local_dir,remote_dir,tmp_local_receive_dir FROM rules`)
					So(err, ShouldBeNil)

					var id int64
					var name, comment, path, loc, rem, tmp string
					var send bool

					So(row.Scan(&id, &name, &send, &comment, &path, &loc, &rem,
						&tmp), ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(name, ShouldEqual, "push")
					So(send, ShouldEqual, true)
					So(comment, ShouldEqual, "this is a comment")
					So(path, ShouldEqual, "/push")
					So(loc, ShouldEqual, "locDir")
					So(rem, ShouldEqual, "remDir")
					So(tmp, ShouldEqual, "tmpDir")
				})
			})
		})
	})
}

func testVer0_7_0RevampTasksTable(eng *testEngine, dialect string) {
	Convey("Given the 0.7.0 'tasks' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampTasksTable{})

		_, err := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		So(err, ShouldBeNil)

		if dialect == migration.PostgreSQL {
			_, err = eng.DB.Exec(`INSERT INTO tasks(rule_id,chain,rank,type,args) 
				VALUES (1,'POST',0,'DELETE',$1)`, []byte("{}"))
		} else {
			_, err = eng.DB.Exec(`INSERT INTO tasks(rule_id,chain,rank,type,args) 
				VALUES (1,'POST',0,'DELETE',?)`, []byte("{}"))
		}

		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampTasksTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT rule_id,chain,rank,type,args FROM tasks`)
				So(err, ShouldBeNil)

				var rID int64
				var rank int16
				var chain, typ, args string

				So(row.Scan(&rID, &chain, &rank, &typ, &args), ShouldBeNil)

				So(rID, ShouldEqual, 1)
				So(chain, ShouldEqual, "POST")
				So(rank, ShouldEqual, 0)
				So(typ, ShouldEqual, "DELETE")
				So(args, ShouldEqual, "{}")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampTasksTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT rule_id,chain,rank,type,args FROM tasks`)
					So(err, ShouldBeNil)

					var rID int64
					var rank int32
					var chain, typ string
					var args []byte

					So(row.Scan(&rID, &chain, &rank, &typ, &args), ShouldBeNil)

					So(rID, ShouldEqual, 1)
					So(chain, ShouldEqual, "POST")
					So(rank, ShouldEqual, 0)
					So(typ, ShouldEqual, "DELETE")
					So(args, ShouldResemble, []byte("{}"))
				})
			})
		})
	})
}

func testVer0_7_0RevampHistoryTable(eng *testEngine) {
	Convey("Given the 0.7.0 'transfer_history' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampHistoryTable{})

		_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            is_send,remote_transfer_id,rule,account,agent,protocol,local_path,
            remote_path,filesize,start,stop,status,step,progression,task_number,
            error_code,error_details) VALUES (1,'waarp_gw',FALSE,TRUE,'abc','push',
            'toto','sftp_part','sftp','/loc/path','/rem/path',123,'2021-01-01T01:00:00.123456Z',
            '2021-01-01T02:00:00.123456Z','CANCELLED','StepData',111,0,'TeDataTransfer',
            'this is an error message')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampHistoryTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				row := eng.DB.QueryRow(`SELECT id,owner,is_server,is_send,
       				remote_transfer_id,rule,account,agent,protocol,local_path,
            		remote_path,filesize,start,stop,status,step,progress,task_number,
           			error_code,error_details FROM transfer_history`)
				So(err, ShouldBeNil)

				var (
					id, prog, size                        int64
					taskNb                                int16
					serv, send                            bool
					start, stop                           time.Time
					owner, remID, rule, acc, ag, proto    string
					lPath, rPath, stat, step, eCode, eDet string
				)

				So(row.Scan(&id, &owner, &serv, &send, &remID, &rule, &acc, &ag,
					&proto, &lPath, &rPath, &size, &start, &stop, &stat, &step,
					&prog, &taskNb, &eCode, &eDet), ShouldBeNil)

				So(id, ShouldEqual, 1)
				So(owner, ShouldEqual, "waarp_gw")
				So(serv, ShouldBeFalse)
				So(send, ShouldBeTrue)
				So(remID, ShouldEqual, "abc")
				So(rule, ShouldEqual, "push")
				So(acc, ShouldEqual, "toto")
				So(ag, ShouldEqual, "sftp_part")
				So(proto, ShouldEqual, "sftp")
				So(lPath, ShouldEqual, "/loc/path")
				So(rPath, ShouldEqual, "/rem/path")
				So(size, ShouldEqual, 123)
				So(start, ShouldHappenWithin, time.Duration(0),
					time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.UTC))
				So(stop, ShouldHappenWithin, time.Duration(0),
					time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.UTC))
				So(stat, ShouldEqual, "CANCELLED") //nolint:misspell //must be kept for retro-compatibility
				So(step, ShouldEqual, "StepData")
				So(prog, ShouldEqual, 111)
				So(taskNb, ShouldEqual, 0)
				So(eCode, ShouldEqual, "TeDataTransfer")
				So(eDet, ShouldEqual, "this is an error message")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampHistoryTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					row := eng.DB.QueryRow(`SELECT id,owner,is_server,is_send,
       				remote_transfer_id,rule,account,agent,protocol,local_path,
            		remote_path,filesize,start,stop,status,step,progression,
       				task_number,error_code,error_details FROM transfer_history`)
					So(err, ShouldBeNil)

					var (
						id, prog, size                        int64
						taskNb                                int16
						serv, send                            bool
						start, stop                           string
						owner, remID, rule, acc, ag, proto    string
						lPath, rPath, stat, step, eCode, eDet string
					)

					So(row.Scan(&id, &owner, &serv, &send, &remID, &rule, &acc, &ag,
						&proto, &lPath, &rPath, &size, &start, &stop, &stat, &step,
						&prog, &taskNb, &eCode, &eDet), ShouldBeNil)

					startTime, err := time.Parse(time.RFC3339, start)
					So(err, ShouldBeNil)

					stopTime, err := time.Parse(time.RFC3339, stop)
					So(err, ShouldBeNil)

					So(id, ShouldEqual, 1)
					So(owner, ShouldEqual, "waarp_gw")
					So(serv, ShouldBeFalse)
					So(send, ShouldBeTrue)
					So(remID, ShouldEqual, "abc")
					So(rule, ShouldEqual, "push")
					So(acc, ShouldEqual, "toto")
					So(ag, ShouldEqual, "sftp_part")
					So(proto, ShouldEqual, "sftp")
					So(lPath, ShouldEqual, "/loc/path")
					So(rPath, ShouldEqual, "/rem/path")
					So(size, ShouldEqual, 123)
					So(startTime, ShouldHappenWithin, time.Duration(0),
						time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.UTC))
					So(stopTime, ShouldHappenWithin, time.Duration(0),
						time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.UTC))
					So(stat, ShouldEqual, "CANCELLED") //nolint:misspell //must be kept for retro-compatibility
					So(step, ShouldEqual, "StepData")
					So(prog, ShouldEqual, 111)
					So(taskNb, ShouldEqual, 0)
					So(eCode, ShouldEqual, "TeDataTransfer")
					So(eDet, ShouldEqual, "this is an error message")
				})
			})
		})
	})
}

func testVer0_7_0RevampTransfersTable(eng *testEngine) {
	Convey("Given the 0.7.0 'transfers' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampTransfersTable{})

		// ### Rule ###
		_, err := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		So(err, ShouldBeNil)

		// ### Local ###
		_, err = eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login) 
			VALUES (100,10,'toto')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,is_server,agent_id,account_id,local_path,remote_path,filesize,
            start,status,step,progression,task_number,error_code,error_details)
            VALUES (1000,'waarp_gw','abcd',1,TRUE,10,100,'/loc/path/1','/rem/path/1',
            1234,'2021-01-01T01:00:00Z','RUNNING','StepData',456,5,'TeDataTransfer',
            'this is an error message')`)
		So(err, ShouldBeNil)

		// ### Remote ###
		_, err = eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login) 
			VALUES (200,20,'toto')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,is_server,agent_id,account_id,local_path,remote_path,filesize,
            start,status,step,progression,task_number,error_code,error_details)
            VALUES (2000,'waarp_gw','efgh',1,FALSE,20,200,'/loc/path/2','/rem/path/2',
            5678,'2021-01-02T01:00:00Z','INTERRUPTED','StepPostTasks',789,8,
            'TeExternalOp','this is another error message')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampTransfersTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				rows, err := eng.DB.Query(`SELECT id,owner,remote_transfer_id,rule_id,
       				local_account_id,remote_account_id,local_path,remote_path,filesize,start,
       				status,step,progress,task_number,error_code,error_details FROM transfers`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						id, ruleID, size, prog                                int64
						lAcc, rAcc                                            sql.NullInt64
						taskNb                                                int16
						owner, remID, lPath, rPath, status, step, eCode, eDet string
						start                                                 time.Time
					)

					So(rows.Scan(&id, &owner, &remID, &ruleID, &lAcc, &rAcc,
						&lPath, &rPath, &size, &start, &status, &step, &prog,
						&taskNb, &eCode, &eDet), ShouldBeNil)

					if id == 1000 {
						So(id, ShouldEqual, 1000)
						So(owner, ShouldEqual, "waarp_gw")
						So(remID, ShouldEqual, "abcd")
						So(ruleID, ShouldEqual, 1)
						So(lAcc.Int64, ShouldEqual, 100)
						So(rAcc.Valid, ShouldBeFalse)
						So(lPath, ShouldEqual, "/loc/path/1")
						So(rPath, ShouldEqual, "/rem/path/1")
						So(size, ShouldEqual, 1234)
						So(start, ShouldHappenWithin, time.Duration(0),
							time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC))
						So(status, ShouldEqual, "RUNNING")
						So(step, ShouldEqual, "StepData")
						So(prog, ShouldEqual, 456)
						So(taskNb, ShouldEqual, 5)
						So(eCode, ShouldEqual, "TeDataTransfer")
						So(eDet, ShouldEqual, "this is an error message")
					} else {
						So(id, ShouldEqual, 2000)
						So(owner, ShouldEqual, "waarp_gw")
						So(remID, ShouldEqual, "efgh")
						So(ruleID, ShouldEqual, 1)
						So(lAcc.Valid, ShouldBeFalse)
						So(rAcc.Int64, ShouldEqual, 200)
						So(lPath, ShouldEqual, "/loc/path/2")
						So(rPath, ShouldEqual, "/rem/path/2")
						So(size, ShouldEqual, 5678)
						So(start, ShouldHappenWithin, time.Duration(0),
							time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC))
						So(status, ShouldEqual, "INTERRUPTED")
						So(step, ShouldEqual, "StepPostTasks")
						So(prog, ShouldEqual, 789)
						So(taskNb, ShouldEqual, 8)
						So(eCode, ShouldEqual, "TeExternalOp")
						So(eDet, ShouldEqual, "this is another error message")
					}
				}
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampTransfersTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					rows, err := eng.DB.Query(`SELECT id,remote_transfer_id,
       					owner,rule_id,is_server,agent_id,account_id,local_path,
       					remote_path,filesize,start,status,step,progression,
       					task_number,error_code,error_details FROM transfers`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							id, ruleID, agID, accID, size, prog, taskNb                  int64
							isServ                                                       bool
							remID, owner, lPath, rPath, start, status, step, eCode, eDet string
						)

						So(rows.Scan(&id, &remID, &owner, &ruleID, &isServ, &agID,
							&accID, &lPath, &rPath, &size, &start, &status, &step,
							&prog, &taskNb, &eCode, &eDet), ShouldBeNil)

						startTime, err := time.Parse(time.RFC3339, start)
						So(err, ShouldBeNil)

						if id == 1000 {
							So(id, ShouldEqual, 1000)
							So(owner, ShouldEqual, "waarp_gw")
							So(remID, ShouldEqual, "abcd")
							So(ruleID, ShouldEqual, 1)
							So(isServ, ShouldBeTrue)
							So(agID, ShouldEqual, 10)
							So(accID, ShouldEqual, 100)
							So(lPath, ShouldEqual, "/loc/path/1")
							So(rPath, ShouldEqual, "/rem/path/1")
							So(size, ShouldEqual, 1234)
							So(startTime.Equal(time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC)), ShouldBeTrue)
							So(status, ShouldEqual, "RUNNING")
							So(step, ShouldEqual, "StepData")
							So(prog, ShouldEqual, 456)
							So(taskNb, ShouldEqual, 5)
							So(eCode, ShouldEqual, "TeDataTransfer")
							So(eDet, ShouldEqual, "this is an error message")
						} else {
							So(id, ShouldEqual, 2000)
							So(owner, ShouldEqual, "waarp_gw")
							So(remID, ShouldEqual, "efgh")
							So(ruleID, ShouldEqual, 1)
							So(isServ, ShouldBeFalse)
							So(agID, ShouldEqual, 20)
							So(accID, ShouldEqual, 200)
							So(lPath, ShouldEqual, "/loc/path/2")
							So(rPath, ShouldEqual, "/rem/path/2")
							So(size, ShouldEqual, 5678)
							So(startTime.Equal(time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC)), ShouldBeTrue)
							So(status, ShouldEqual, "INTERRUPTED")
							So(step, ShouldEqual, "StepPostTasks")
							So(prog, ShouldEqual, 789)
							So(taskNb, ShouldEqual, 8)
							So(eCode, ShouldEqual, "TeExternalOp")
							So(eDet, ShouldEqual, "this is another error message")
						}
					}
				})
			})
		})
	})
}

func testVer0_7_0RevampTransferInfoTable(eng *testEngine) {
	Convey("Given the 0.7.0 'transfer_info' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampTransferInfoTable{})

		_, err := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login) 
			VALUES (100,10,'toto')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,local_account_id,remote_account_id,local_path,remote_path,
            filesize,start,status,step,progress,task_number,error_code,error_details)
            VALUES (1000,'waarp_gw','abcd',1,100,null,'/loc/path/1','/rem/path/1',
            1234,'2021-01-01 01:00:00','RUNNING','StepData',456,5,'TeDataTransfer',
            'this is an error message')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            is_send,remote_transfer_id,rule,account,agent,protocol,local_path,
            remote_path,filesize,start,stop,status,step,progress,task_number,
            error_code,error_details) VALUES (2000,'waarp_gw',FALSE,TRUE,'abc','push',
            'toto','sftp_part','sftp','/loc/path','/rem/path',123,'2021-01-01 01:00:00',
            '2021-01-01 02:00:00','CANCELLED','StepData',111,0,'TeDataTransfer',
            'this is an error message')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfer_info(transfer_id,is_history,
        	name,value) VALUES (1000,false,'t_info','"t_value"')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfer_info(transfer_id,is_history,
        	name,value) VALUES (2000,true,'h_info','"h_value"')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampTransferInfoTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				rows, err := eng.DB.Query(`SELECT transfer_id,history_id,name,value
					FROM transfer_info`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var tID, hID sql.NullInt64
					var name, value string

					So(rows.Scan(&tID, &hID, &name, &value), ShouldBeNil)

					if tID.Valid {
						So(tID.Int64, ShouldEqual, 1000)
						So(hID.Int64, ShouldEqual, 0)
						So(name, ShouldEqual, "t_info")
						So(value, ShouldEqual, `"t_value"`)
					} else {
						So(tID.Int64, ShouldEqual, 0)
						So(hID.Int64, ShouldEqual, 2000)
						So(name, ShouldEqual, "h_info")
						So(value, ShouldEqual, `"h_value"`)
					}
				}
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampTransferInfoTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
				})
			})
		})
	})
}

func testVer0_7_0RevampCryptoTable(eng *testEngine) {
	Convey("Given the 0.7.0 'crypto' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampCryptoTable{})

		// ### Local agent ###
		_, err := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		// ### Local account ###
		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login) 
			VALUES (100,10,'toto')`)
		So(err, ShouldBeNil)

		// ### Remote agent ###
		_, err = eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		So(err, ShouldBeNil)

		// ### Remote account ###
		_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login) 
			VALUES (200,20,'toto')`)
		So(err, ShouldBeNil)

		// ##### Certificates #####
		_, err = eng.DB.Exec(`INSERT INTO crypto_credentials(id,name,owner_type,
            owner_id,private_key,certificate,ssh_public_key) VALUES (1010,
            'lag_cert','local_agents',10,'pk1','cert1','pbk1')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO crypto_credentials(id,name,owner_type,
            owner_id,private_key,certificate,ssh_public_key) VALUES (1100,
        	'lac_cert','local_accounts',100,'pk2','cert2','pbk2')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO crypto_credentials(id,name,owner_type,
            owner_id,private_key,certificate,ssh_public_key) VALUES (2020,
            'rag_cert','remote_agents',20,'pk3','cert3','pbk3')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO crypto_credentials(id,name,owner_type,
            owner_id,private_key,certificate,ssh_public_key) VALUES (2200,
            'rac_cert','remote_accounts',200,'pk4','cert4','pbk4')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampCryptoTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				rows, err := eng.DB.Query(`SELECT id,name,local_agent_id,
       				remote_agent_id,local_account_id,remote_account_id,
       				private_key,certificate,ssh_public_key FROM crypto_credentials`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						id                         int64
						lagID, lacID, ragID, racID sql.NullInt64
						name, pk, cert, pbk        string
					)

					So(rows.Scan(&id, &name, &lagID, &ragID, &lacID, &racID,
						&pk, &cert, &pbk), ShouldBeNil)

					switch {
					case lagID.Valid:
						So(id, ShouldEqual, 1010)
						So(name, ShouldEqual, "lag_cert")
						So(lagID.Int64, ShouldEqual, 10)
						So(ragID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 0)
						So(pk, ShouldEqual, "pk1")
						So(cert, ShouldEqual, "cert1")
						So(pbk, ShouldEqual, "pbk1")
					case lacID.Valid:
						So(id, ShouldEqual, 1100)
						So(name, ShouldEqual, "lac_cert")
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 100)
						So(ragID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 0)
						So(pk, ShouldEqual, "pk2")
						So(cert, ShouldEqual, "cert2")
						So(pbk, ShouldEqual, "pbk2")
					case ragID.Valid:
						So(id, ShouldEqual, 2020)
						So(name, ShouldEqual, "rag_cert")
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(ragID.Int64, ShouldEqual, 20)
						So(racID.Int64, ShouldEqual, 0)
						So(pk, ShouldEqual, "pk3")
						So(cert, ShouldEqual, "cert3")
						So(pbk, ShouldEqual, "pbk3")
					case racID.Valid:
						So(id, ShouldEqual, 2200)
						So(name, ShouldEqual, "rac_cert")
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(ragID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 200)
						So(pk, ShouldEqual, "pk4")
						So(cert, ShouldEqual, "cert4")
						So(pbk, ShouldEqual, "pbk4")
					default:
						panic("crypto is missing an owner")
					}
				}
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampCryptoTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					rows, err := eng.DB.Query(`SELECT id,name,owner_type,owner_id,
       					private_key,certificate,ssh_public_key FROM crypto_credentials`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							id, oID                    int64
							name, oType, pk, cert, pbk string
						)

						So(rows.Scan(&id, &name, &oType, &oID, &pk, &cert, &pbk), ShouldBeNil)

						switch oType {
						case "local_agents":
							So(id, ShouldEqual, 1010)
							So(name, ShouldEqual, "lag_cert")
							So(oID, ShouldEqual, 10)
							So(pk, ShouldEqual, "pk1")
							So(cert, ShouldEqual, "cert1")
							So(pbk, ShouldEqual, "pbk1")
						case "local_accounts":
							So(id, ShouldEqual, 1100)
							So(name, ShouldEqual, "lac_cert")
							So(oID, ShouldEqual, 100)
							So(pk, ShouldEqual, "pk2")
							So(cert, ShouldEqual, "cert2")
							So(pbk, ShouldEqual, "pbk2")
						case "remote_agents":
							So(id, ShouldEqual, 2020)
							So(name, ShouldEqual, "rag_cert")
							So(oID, ShouldEqual, 20)
							So(pk, ShouldEqual, "pk3")
							So(cert, ShouldEqual, "cert3")
							So(pbk, ShouldEqual, "pbk3")
						case "remote_accounts":
							So(id, ShouldEqual, 2200)
							So(name, ShouldEqual, "rac_cert")
							So(oID, ShouldEqual, 200)
							So(pk, ShouldEqual, "pk4")
							So(cert, ShouldEqual, "cert4")
							So(pbk, ShouldEqual, "pbk4")
						default:
							panic(fmt.Sprintf("unknown crypto owner type '%s'", oType))
						}
					}
				})
			})
		})
	})
}

func testVer0_7_0RevampRuleAccessTable(eng *testEngine) {
	Convey("Given the 0.7.0 'rule_access' table revamp", func() {
		setupDatabaseUpTo(eng, ver0_7_0RevampRuleAccessTable{})

		// ### Rule ###
		_, err := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		So(err, ShouldBeNil)

		// ### Local agent ###
		_, err = eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config) 
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		So(err, ShouldBeNil)

		// ### Local account ###
		_, err = eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login) 
			VALUES (100,10,'toto')`)
		So(err, ShouldBeNil)

		// ### Remote agent ###
		_, err = eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		So(err, ShouldBeNil)

		// ### Remote account ###
		_, err = eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login) 
			VALUES (200,20,'toto')`)
		So(err, ShouldBeNil)

		// ##### Accesses #####
		_, err = eng.DB.Exec(`INSERT INTO rule_access(rule_id,object_type,object_id) 
			VALUES (1,'local_agents',10)`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rule_access(rule_id,object_type,object_id) 
			VALUES (1,'local_accounts',100)`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rule_access(rule_id,object_type,object_id) 
			VALUES (1,'remote_agents',20)`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rule_access(rule_id,object_type,object_id) 
			VALUES (1,'remote_accounts',200)`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade(ver0_7_0RevampRuleAccessTable{})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the columns", func() {
				rows, err := eng.DB.Query(`SELECT rule_id,local_agent_id,
       				remote_agent_id,local_account_id,remote_account_id
       				FROM rule_access`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						ruleID                     int64
						lagID, lacID, ragID, racID sql.NullInt64
					)

					So(rows.Scan(&ruleID, &lagID, &ragID, &lacID, &racID), ShouldBeNil)

					switch {
					case lagID.Valid:
						So(ruleID, ShouldEqual, 1)
						So(lagID.Int64, ShouldEqual, 10)
						So(ragID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 0)
					case lacID.Valid:
						So(ruleID, ShouldEqual, 1)
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 100)
						So(ragID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 0)
					case ragID.Valid:
						So(ruleID, ShouldEqual, 1)
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(ragID.Int64, ShouldEqual, 20)
						So(racID.Int64, ShouldEqual, 0)
					case racID.Valid:
						So(ruleID, ShouldEqual, 1)
						So(lagID.Int64, ShouldEqual, 0)
						So(lacID.Int64, ShouldEqual, 0)
						So(ragID.Int64, ShouldEqual, 0)
						So(racID.Int64, ShouldEqual, 200)
					default:
						panic("rule access is missing a target")
					}
				}
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade(ver0_7_0RevampRuleAccessTable{})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the column changes", func() {
					rows, err := eng.DB.Query(`SELECT rule_id,object_type,
       					object_id FROM rule_access`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							ruleID, oID int64
							oType       string
						)

						So(rows.Scan(&ruleID, &oType, &oID), ShouldBeNil)

						switch oType {
						case "local_agents":
							So(ruleID, ShouldEqual, 1)
							So(oID, ShouldEqual, 10)
						case "local_accounts":
							So(ruleID, ShouldEqual, 1)
							So(oID, ShouldEqual, 100)
						case "remote_agents":
							So(ruleID, ShouldEqual, 1)
							So(oID, ShouldEqual, 20)
						case "remote_accounts":
							So(ruleID, ShouldEqual, 1)
							So(oID, ShouldEqual, 200)
						default:
							panic(fmt.Sprintf("unknown rule access object type '%s'", oType))
						}
					}
				})
			})
		})
	})
}
