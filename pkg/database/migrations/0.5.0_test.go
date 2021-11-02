package migrations

import (
	"path/filepath"
	"runtime"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

func testVer0_5_0LocalAgentChangePaths(eng *migration.Engine) {
	Convey("Given the 0.5.0 local agent paths change", func() {
		setupDatabaseUpTo(eng, ver0_5_0LocalAgentChangePaths{})

		_, err := eng.DB.Exec(`INSERT INTO local_agents (owner,name,protocol,
            proto_config,address,root,in_dir,out_dir,work_dir) 
			VALUES ('test_gw','toto','test_proto','{}','[::1]:1','/C:/root',
			'files/in','files/out','files/work')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0LocalAgentChangePaths{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the paths", func() {
				row := eng.DB.QueryRow(`SELECT root,in_dir,out_dir,work_dir FROM local_agents`)

				var root, in, out, tmp string
				So(row.Scan(&root, &in, &out, &tmp), ShouldBeNil)

				if runtime.GOOS == windowsRuntime {
					So(root, ShouldEqual, filepath.FromSlash("C:/root"))
				} else {
					So(root, ShouldEqual, filepath.FromSlash("/C:/root"))
				}

				So(in, ShouldEqual, filepath.FromSlash("files/in"))
				So(out, ShouldEqual, filepath.FromSlash("files/out"))
				So(tmp, ShouldEqual, filepath.FromSlash("files/work"))
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0LocalAgentChangePaths{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have restored the paths", func() {
					row := eng.DB.QueryRow(`SELECT root,in_dir,out_dir,work_dir FROM local_agents`)

					var root, in, out, tmp string
					So(row.Scan(&root, &in, &out, &tmp), ShouldBeNil)

					So(root, ShouldEqual, "/C:/root")
					So(in, ShouldEqual, "files/in")
					So(out, ShouldEqual, "files/out")
					So(tmp, ShouldEqual, "files/work")
				})
			})
		})
	})
}

func testVer0_5_0LocalAgentDisallowReservedNames(eng *migration.Engine) {
	Convey("Given the 0.5.0 local agent name verification", func() {
		setupDatabaseUpTo(eng, ver0_5_0LocalAgentChangePaths{})

		_, err := eng.DB.Exec(`INSERT INTO local_agents (
            owner,name,protocol,proto_config,address,root,in_dir,out_dir,work_dir) 
			VALUES ('test_gw','toto','test_proto','{}','[::1]:1','','','','')`)
		So(err, ShouldBeNil)

		Convey("Given that all server names are valid", func() {
			Convey("When applying the migration", func() {
				err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0LocalAgentDisallowReservedNames{}}})

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that some server names are reserved", func() {
			_, err := eng.DB.Exec(`INSERT INTO local_agents (
            owner,name,protocol,proto_config,address,root,in_dir,out_dir,work_dir)   
			VALUES ('test_gw','Database','test_proto','{}','[::1]:2','','','','')`)
			So(err, ShouldBeNil)

			Convey("When applying the migration", func() {
				err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0LocalAgentDisallowReservedNames{}}})

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, "'Database' is a reserved service name, "+
						"this server should be renamed")
				})
			})
		})
	})
}

func testVer0_5_0RuleNewPathCols(eng *migration.Engine) {
	Convey("Given the 0.5.0 rule new path columns addition", func() {
		setupDatabaseUpTo(eng, ver0_5_0RuleNewPathCols{})

		_, err := eng.DB.Exec(`INSERT INTO rules (
            name,send,comment,path,in_path,out_path,work_path) 
			VALUES ('snd',true,'','/snd_path','in','out','tmp')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rules (
            name,send,comment,path,in_path,out_path,work_path) 
			VALUES ('rcv',false,'','/rcv_path','in','out','tmp')`)
		So(err, ShouldBeNil)

		tableShouldHaveColumn(eng.DB, "rules", "in_path", "out_path", "work_path")

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0RuleNewPathCols{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have added the new columns", func() {
				tableShouldNotHaveColumn(eng.DB, "rules", "in_path", "out_path", "work_path")
				tableShouldHaveColumn(eng.DB, "rules", "local_dir", "remote_dir", "local_tmp_dir")

				Convey("Then the columns should have been filled", func() {
					rows, err := eng.DB.Query(`SELECT send, local_dir,remote_dir,
       					local_tmp_dir FROM rules`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							send          bool
							loc, rem, tmp string
						)
						So(rows.Scan(&send, &loc, &rem, &tmp), ShouldBeNil)

						if send {
							So(loc, ShouldEqual, "out")
							So(rem, ShouldEqual, "in")
						} else {
							So(loc, ShouldEqual, "in")
							So(rem, ShouldEqual, "out")
						}

						So(tmp, ShouldEqual, "tmp")
					}
				})
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0RuleNewPathCols{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have dropped the new column", func() {
					tableShouldHaveColumn(eng.DB, "rules", "in_path", "out_path", "work_path")
					tableShouldNotHaveColumn(eng.DB, "rules", "local_dir",
						"remote_dir", "local_tmp_dir")
				})
			})
		})
	})
}

func testVer0_5_0RulePathChanges(eng *migration.Engine) {
	Convey("Given the 0.5.0 rule paths change", func() {
		setupDatabaseUpTo(eng, ver0_5_0RulePathChanges{})

		_, err := eng.DB.Exec(`INSERT INTO rules (
            name,send,comment,path,local_dir,remote_dir,local_tmp_dir) VALUES 
            ('snd',true,'','/snd_path','local/dir','remote/dir','/C:/tmp/dir')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0RulePathChanges{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the paths", func() {
				row := eng.DB.QueryRow("SELECT local_dir,remote_dir,local_tmp_dir FROM rules")

				var loc, rem, tmp string
				So(row.Scan(&loc, &rem, &tmp), ShouldBeNil)

				So(loc, ShouldEqual, filepath.FromSlash("local/dir"))
				So(rem, ShouldEqual, "remote/dir")

				if runtime.GOOS == windowsRuntime {
					So(tmp, ShouldEqual, filepath.FromSlash("C:/tmp/dir"))
				} else {
					So(tmp, ShouldEqual, filepath.FromSlash("/C:/tmp/dir"))
				}
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0RulePathChanges{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have restored the paths", func() {
					row := eng.DB.QueryRow("SELECT local_dir,remote_dir,local_tmp_dir FROM rules")

					var loc, rem, tmp string
					So(row.Scan(&loc, &rem, &tmp), ShouldBeNil)

					So(loc, ShouldEqual, "local/dir")
					So(rem, ShouldEqual, "remote/dir")
					So(tmp, ShouldEqual, "/C:/tmp/dir")
				})
			})
		})
	})
}

func testVer0_5_0AddFilesize(eng *migration.Engine) {
	Convey("Given the 0.5.0 filesize addition", func() {
		setupDatabaseUpTo(eng, ver0_5_0AddFilesize{})
		tableShouldNotHaveColumn(eng.DB, "transfers", "filesize")
		tableShouldNotHaveColumn(eng.DB, "transfer_history", "filesize")

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0AddFilesize{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have added the columns", func() {
				tableShouldHaveColumn(eng.DB, "transfers", "filesize")
				tableShouldHaveColumn(eng.DB, "transfer_history", "filesize")
			})

			Convey("When reversing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0AddFilesize{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have removed the columns", func() {
					tableShouldNotHaveColumn(eng.DB, "transfers", "filesize")
					tableShouldNotHaveColumn(eng.DB, "transfer_history", "filesize")
				})
			})
		})
	})
}

func testVer0_5_0TransferChangePaths(eng *migration.Engine) {
	Convey("Given the 0.5.0 transfer paths changes", func() {
		setupDatabaseUpTo(eng, ver0_5_0TransferChangePaths{})

		_, err := eng.DB.Exec(`INSERT INTO rules (
            id,name,send,comment,path,local_dir,remote_dir,local_tmp_dir)
			VALUES (1,'snd',true,'','/snd_path','snd_loc','snd_rem','snd_tmp')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO rules (
            id,name,send,comment,path,local_dir,remote_dir,local_tmp_dir)
			VALUES (2,'rcv',false,'','/rcv_path','rcv_loc','rcv_rem','rcv_tmp')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers (owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,true_filepath,source_file,dest_file,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			('', '', false,1,0,0,'/C:/src1','src1','dst1','1999-01-08 04:05:06 -8:00',
			 '','',0,0,'','')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfers (owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,true_filepath,source_file,dest_file,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			('', '', false,2,0,0,'/C:/dst2','src2','dst2','1999-01-08 04:05:06 -8:00',
			 '','',0,0,'','')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0TransferChangePaths{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have renamed the local path column", func() {
				tableShouldHaveColumn(eng.DB, "transfers", "local_path")
				tableShouldNotHaveColumn(eng.DB, "transfers", "true_filepath")
			})

			Convey("Then it should have replaced the source & dest file columns", func() {
				tableShouldHaveColumn(eng.DB, "transfers", "remote_path")
				tableShouldNotHaveColumn(eng.DB, "transfers", "source_file", "dest_file")

				rows, err := eng.DB.Query(`SELECT transfers.remote_path, rules.send
					FROM transfers INNER JOIN rules ON transfers.rule_id=rules.id`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						path string
						send bool
					)
					So(rows.Scan(&path, &send), ShouldBeNil)

					if send {
						So(path, ShouldEqual, "snd_rem/dst1")
					} else {
						So(path, ShouldEqual, "rcv_rem/src2")
					}
				}
			})

			Convey("When reverting the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0TransferChangePaths{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the changes", func() {
					rows, err := eng.DB.Query(`SELECT transfers.true_filepath,
       					transfers.source_file,transfers.dest_file,rules.send FROM 
       					transfers INNER JOIN rules ON transfers.rule_id=rules.id`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var (
							full, src, dst string
							send           bool
						)

						So(rows.Scan(&full, &src, &dst, &send), ShouldBeNil)

						if send {
							So(src, ShouldEqual, "src1")
							So(dst, ShouldEqual, "dst1")
							So(full, ShouldEqual, "/C:/"+src)
						} else {
							So(src, ShouldEqual, "src2")
							So(dst, ShouldEqual, "dst2")
							So(full, ShouldEqual, "/C:/"+dst)
						}
					}
				})
			})
		})
	})
}

func testVer0_5_0TransferFormatLocalPath(eng *migration.Engine) {
	Convey("Given the 0.5.0 transfer local path formatting", func() {
		setupDatabaseUpTo(eng, ver0_5_0TransferFormatLocalPath{})

		_, err := eng.DB.Exec(`INSERT INTO transfers (owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,local_path,remote_path,start,
			status,step,progression,task_number,error_code,error_details) VALUES
			('', '', false,1,0,0,'/C:/src','remote/dst','1999-01-08 04:05:06 -8:00',
			 '','',0,0,'','')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0TransferFormatLocalPath{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have formatted the local path to the OS format", func() {
				row := eng.DB.QueryRow(`SELECT local_path FROM transfers`)

				var path string
				So(row.Scan(&path), ShouldBeNil)

				if runtime.GOOS == windowsRuntime {
					So(path, ShouldStartWith, filepath.FromSlash("C:/src"))
				} else {
					So(path, ShouldStartWith, filepath.FromSlash("/C:/src"))
				}
			})

			Convey("When undoing the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0TransferFormatLocalPath{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the local path to a URI", func() {
					row := eng.DB.QueryRow(`SELECT local_path FROM transfers`)

					var path string
					So(row.Scan(&path), ShouldBeNil)

					So(path, ShouldStartWith, "/C:/src")
				})
			})
		})
	})
}

func testVer0_5_0HistoryChangePaths(eng *migration.Engine) {
	Convey("Given the 0.5.0 history paths changes", func() {
		setupDatabaseUpTo(eng, ver0_5_0HistoryPathsChange{})

		_, err := eng.DB.Exec(`INSERT INTO transfer_history (id,owner,remote_transfer_id,
        	protocol,is_server,is_send,rule,agent,account,source_filename,dest_filename,
			start,stop,status,step,progression,task_number,error_code,error_details) VALUES
			(1,'','','',false,true,'','','','src_file','dst_file','1999-01-08 04:05:06 -8:00',
			 '1999-01-08 04:05:06 -8:00','','',0,0,'','')`)
		So(err, ShouldBeNil)

		_, err = eng.DB.Exec(`INSERT INTO transfer_history (id,owner,remote_transfer_id,
        	protocol,is_server,is_send,rule,agent,account,source_filename,dest_filename,
			start,stop,status,step,progression,task_number,error_code,error_details) VALUES
			(2,'','','',false,false,'','','','src_file','dst_file','1999-01-08 04:05:06 -8:00',
			 '1999-01-08 04:05:06 -8:00','','',0,0,'','')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0HistoryPathsChange{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have renamed the filename columns", func() {
				tableShouldHaveColumn(eng.DB, "transfer_history", "local_path", "remote_path")
				tableShouldNotHaveColumn(eng.DB, "transfers", "source_filename", "dest_filename")

				rows, err := eng.DB.Query(`SELECT is_send,local_path,remote_path FROM transfer_history`)
				So(err, ShouldBeNil)
				So(rows.Err(), ShouldBeNil)

				defer rows.Close()

				for rows.Next() {
					var (
						send     bool
						loc, rem string
					)

					So(rows.Scan(&send, &loc, &rem), ShouldBeNil)

					if send {
						So(loc, ShouldEqual, "src_file")
						So(rem, ShouldEqual, "dst_file")
					} else {
						So(loc, ShouldEqual, "dst_file")
						So(rem, ShouldEqual, "src_file")
					}
				}
			})

			Convey("When reverting the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0HistoryPathsChange{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the changes", func() {
					tableShouldNotHaveColumn(eng.DB, "transfer_history", "local_path", "remote_path")
					tableShouldHaveColumn(eng.DB, "transfer_history", "source_filename", "dest_filename")

					rows, err := eng.DB.Query(`SELECT source_filename,dest_filename FROM transfer_history`)
					So(err, ShouldBeNil)
					So(rows.Err(), ShouldBeNil)

					defer rows.Close()

					for rows.Next() {
						var src, dst string

						So(rows.Scan(&src, &dst), ShouldBeNil)

						So(src, ShouldEqual, "src_file")
						So(dst, ShouldEqual, "dst_file")
					}
				})
			})
		})
	})
}

func testVer0_5_0LocalAccountsPasswordDecode(eng *migration.Engine) {
	Convey("Given the 0.5.0 local accounts password decoding", func() {
		setupDatabaseUpTo(eng, ver0_5_0LocalAccountsPasswordDecode{})

		_, err := eng.DB.Exec(`INSERT INTO local_accounts (id,local_agent_id,
				login,password_hash) VALUES (1,'1','toto','Zm9vYmFy')`)
		So(err, ShouldBeNil)

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0LocalAccountsPasswordDecode{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have decoded the password hash", func() {
				row := eng.DB.QueryRow(`SELECT password_hash FROM local_accounts`)

				var hash string
				So(row.Scan(&hash), ShouldBeNil)

				So(hash, ShouldEqual, "foobar")
			})

			Convey("When reverting the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0LocalAccountsPasswordDecode{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have re-encoded the password hash", func() {
					row := eng.DB.QueryRow(`SELECT password_hash FROM local_accounts`)

					var hash string
					So(row.Scan(&hash), ShouldBeNil)

					So(hash, ShouldEqual, "Zm9vYmFy")
				})
			})
		})
	})
}

func testVer0_5_0UserPasswordChange(eng *migration.Engine, dialect string) {
	Convey("Given the 0.5.0 user password changes", func() {
		setupDatabaseUpTo(eng, ver0_5_0UserPasswordChange{})

		pswd := []byte("Zm9vYmFy")
		perm := []byte{0b10101010}

		if dialect == migration.PostgreSQL {
			_, err := eng.DB.Exec(`INSERT INTO users (owner,id,username,password,
			   permissions)	VALUES ('gw',1,'toto',$1,$2)`, pswd, perm)
			So(err, ShouldBeNil)
		} else {
			_, err := eng.DB.Exec(`INSERT INTO users (owner,id,username,password,
			   permissions)	VALUES ('gw',1,'toto',?,?)`, pswd, perm)
			So(err, ShouldBeNil)
		}

		Convey("When applying the migration", func() {
			err := eng.Upgrade([]migration.Migration{{Script: ver0_5_0UserPasswordChange{}}})
			So(err, ShouldBeNil)

			Convey("Then it should have changed the password column", func() {
				tableShouldNotHaveColumn(eng.DB, "users", "password")
				tableShouldHaveColumn(eng.DB, "users", "password_hash")

				row := eng.DB.QueryRow(`SELECT password_hash FROM users`)

				var hash string
				So(row.Scan(&hash), ShouldBeNil)

				So(hash, ShouldEqual, "Zm9vYmFy")
			})

			Convey("When reverting the migration", func() {
				err := eng.Downgrade([]migration.Migration{{Script: ver0_5_0UserPasswordChange{}}})
				So(err, ShouldBeNil)

				Convey("Then it should have reverted the password changes", func() {
					tableShouldNotHaveColumn(eng.DB, "users", "password_hash")
					tableShouldHaveColumn(eng.DB, "users", "password")

					row := eng.DB.QueryRow(`SELECT password FROM users`)

					var hash []byte
					So(row.Scan(&hash), ShouldBeNil)

					So(hash, ShouldResemble, []byte("Zm9vYmFy"))
				})
			})
		})
	})
}
