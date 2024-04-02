package migrations

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_5_0RemoveRulePathSlash(t *testing.T, eng *testEngine) Change {
	mig := Migrations[2]

	t.Run("When applying the 0.5.0 rule slash removal change", func(t *testing.T) {
		query := `INSERT INTO rules (name, comment, send, path, in_path, 
            out_path, work_path) VALUES (?, ?, ?, ?, ?, ?, ?)`

		eng.NoError(t, query, "send", "", true, "/send_path", "", "", "")
		eng.NoError(t, query, "recv", "", false, "/recv_path", "", "", "")

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM rules")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have removed the leading slash from rules' paths", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT path FROM rules ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var path1, path2 string
			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&path1))
			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&path2))

			assert.Equal(t, "send_path", path1)
			assert.Equal(t, "recv_path", path2)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the leading slash", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT path FROM rules ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var path1, path2 string
				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&path1))
				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&path2))

				assert.Equal(t, "/send_path", path1)
				assert.Equal(t, "/recv_path", path2)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_5_0CheckRulePathAncestor(t *testing.T, eng *testEngine) Change {
	mig := Migrations[3]

	t.Run("When applying the 0.5.0 rule path ancestry check", func(t *testing.T) {
		query := `INSERT INTO rules (name, comment, send, path, in_path, 
            out_path, work_path) VALUES (?, ?, ?, ?, ?, ?, ?)`

		eng.NoError(t, query, "send", "", true, "dir/send_path", "", "", "")
		eng.NoError(t, query, "recv", "", false, "dir/recv_path", "", "", "")

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM rules")
		})

		t.Run("Given that a rule path IS another one's parent", func(t *testing.T) {
			query2 := `UPDATE rules SET path=? WHERE name=?`
			eng.NoError(t, query2, "dir", "recv")

			t.Cleanup(func() {
				eng.NoError(t, query2, "dir/send_path", "recv")
			})

			require.EqualError(t,
				eng.Upgrade(mig),
				"the path of the rule 'recv' (dir) must be changed so that it is "+
					"no longer a parent of the path of rule 'send' (dir/send_path)",
				"The migration should fail")
		})

		t.Run("Given that no rule path is NOT another one's parent", func(t *testing.T) {
			require.NoError(t, eng.Upgrade(mig),
				"The migration should not fail")
		})
	})

	return mig
}

func testVer0_5_0LocalAgentChangePaths(t *testing.T, eng *testEngine) Change {
	mig := Migrations[4]

	t.Run("When applying the 0.5.0 local agent paths change", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO local_agents (owner,name,protocol,
            proto_config,address,root,in_dir,out_dir,work_dir)
			VALUES ('test_gw','toto','test_proto','{}','[::1]:1','/C:/root',
			'files/in','files/out','files/work')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM local_agents")
		})

		require.NoError(t, eng.Upgrade(mig),
			"the migration should not fail")

		t.Run("Then it should have changed the paths", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT root,in_dir,out_dir,work_dir FROM local_agents`)

			var root, in, out, tmp string
			require.NoError(t, row.Scan(&root, &in, &out, &tmp))

			if isWindowsRuntime() {
				assert.Equal(t, filepath.FromSlash("C:/root"), root)
			} else {
				assert.Equal(t, filepath.FromSlash("/C:/root"), root)
			}

			assert.Equal(t, filepath.FromSlash("files/in"), in)
			assert.Equal(t, filepath.FromSlash("files/out"), out)
			assert.Equal(t, filepath.FromSlash("files/work"), tmp)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have re-normalized the paths", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT root,in_dir,out_dir,work_dir FROM local_agents`)

				var root, in, out, tmp string
				require.NoError(t, row.Scan(&root, &in, &out, &tmp))

				assert.Equal(t, "/C:/root", root)
				assert.Equal(t, "files/in", in)
				assert.Equal(t, "files/out", out)
				assert.Equal(t, "files/work", tmp)
			})
		})
	})

	return mig
}

func testVer0_5_0LocalAgentsPathsRename(t *testing.T, eng *testEngine) Change {
	mig := Migrations[5]

	t.Run("When applying the 0.5.0 local agent path column rename", func(t *testing.T) {
		tableShouldHaveColumns(t, eng.DB, "local_agents", "root", "in_dir", "out_dir", "work_dir")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have renamed the path columns", func(t *testing.T) {
			tableShouldNotHaveColumns(t, eng.DB, "local_agents", "root",
				"in_dir", "out_dir", "work_dir")
			tableShouldHaveColumns(t, eng.DB, "local_agents", "root_dir",
				"receive_dir", "send_dir", "tmp_receive_dir")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the old column names", func(t *testing.T) {
				tableShouldHaveColumns(t, eng.DB, "local_agents", "root", "in_dir",
					"out_dir", "work_dir")
				tableShouldNotHaveColumns(t, eng.DB, "local_agents", "root_dir",
					"receive_dir", "send_dir", "tmp_receive_dir")
			})
		})
	})

	return mig
}

func testVer0_5_0LocalAgentDisallowReservedNames(t *testing.T, eng *testEngine) Change {
	mig := Migrations[6]

	t.Run("When applying the 0.5.0 local agent name verification", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO local_agents (owner,name,protocol,
				proto_config,address,root_dir,send_dir,receive_dir,tmp_receive_dir)
			VALUES ('test_gw','toto','test_proto','{}','[::1]:1','','','','')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM local_agents")
		})

		t.Run("Given that some server names are reserved", func(t *testing.T) {
			eng.NoError(t, `INSERT INTO local_agents (owner,name,protocol,
				proto_config,address,root_dir,send_dir,receive_dir,tmp_receive_dir)
			VALUES ('test_gw','Database','test_proto','{}','[::1]:2','','','','')`)

			t.Cleanup(func() {
				eng.NoError(t, `DELETE FROM local_agents WHERE name='Database'`)
			})

			require.EqualError(t,
				eng.Upgrade(mig),
				`"Database" is a reserved service name, this server should be renamed`,
				"The migration should fail")
		})

		t.Run("Given that all server names are valid", func(t *testing.T) {
			require.NoError(t, eng.Upgrade(mig),
				"The migration should not fail")
		})
	})

	return mig
}

func testVer0_5_0RuleNewPathCols(t *testing.T, eng *testEngine) Change {
	mig := Migrations[7]

	t.Run("When applying the 0.5.0 rule new path columns addition", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO rules(id,name,send,comment,path,in_path,out_path,work_path)
			VALUES (1,'snd',true,'','/snd_path','in','out','tmp'),
			       (2,'rcv',false,'','/rcv_path','in','out','tmp')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM rules`)
		})

		tableShouldHaveColumns(t, eng.DB, "rules", "in_path", "out_path", "work_path")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the new columns", func(t *testing.T) {
			tableShouldNotHaveColumns(t, eng.DB, "rules", "in_path", "out_path", "work_path")
			tableShouldHaveColumns(t, eng.DB, "rules", "local_dir", "remote_dir", "tmp_local_receive_dir")

			t.Run("Then the columns should have been filled", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT send, local_dir,remote_dir,
       					tmp_local_receive_dir FROM rules ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					send          bool
					loc, rem, tmp string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&send, &loc, &rem, &tmp))
				assert.True(t, send)
				assert.Equal(t, "out", loc)
				assert.Equal(t, "in", rem)
				assert.Equal(t, "tmp", tmp)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&send, &loc, &rem, &tmp))
				assert.False(t, send)
				assert.Equal(t, "in", loc)
				assert.Equal(t, "out", rem)
				assert.Equal(t, "tmp", tmp)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new column", func(t *testing.T) {
				tableShouldHaveColumns(t, eng.DB, "rules", "in_path", "out_path", "work_path")
				tableShouldNotHaveColumns(t, eng.DB, "rules", "local_dir", "remote_dir", "tmp_local_receive_dir")
			})
		})
	})

	return mig
}

func testVer0_5_0RulePathChanges(t *testing.T, eng *testEngine) Change {
	mig := Migrations[8]

	t.Run("When applying the 0.5.0 rule paths change", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO rules (
            name,send,comment,path,local_dir,remote_dir,tmp_local_receive_dir) VALUES
            ('snd',true,'','/snd_path','local/dir','remote/dir','/C:/tmp/dir')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM rules`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the paths", func(t *testing.T) {
			row := eng.DB.QueryRow("SELECT local_dir,remote_dir,tmp_local_receive_dir FROM rules")

			var loc, rem, tmp string
			require.NoError(t, row.Scan(&loc, &rem, &tmp))

			assert.Equal(t, filepath.FromSlash("local/dir"), loc)
			assert.Equal(t, "remote/dir", rem)

			if isWindowsRuntime() {
				assert.Equal(t, `C:\tmp\dir`, tmp)
			} else {
				assert.Equal(t, "/C:/tmp/dir", tmp)
			}
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the paths", func(t *testing.T) {
				row := eng.DB.QueryRow("SELECT local_dir,remote_dir,tmp_local_receive_dir FROM rules")

				var loc, rem, tmp string
				require.NoError(t, row.Scan(&loc, &rem, &tmp))

				assert.Equal(t, "local/dir", loc)
				assert.Equal(t, "remote/dir", rem)
				assert.Equal(t, "/C:/tmp/dir", tmp)
			})
		})
	})

	return mig
}

func testVer0_5_0AddFilesize(t *testing.T, eng *testEngine) Change {
	mig := Migrations[9]

	t.Run("When applying the 0.5.0 filesize addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "transfers", "filesize")
		tableShouldNotHaveColumns(t, eng.DB, "transfer_history", "filesize")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "filesize")
			tableShouldHaveColumns(t, eng.DB, "transfer_history", "filesize")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have removed the column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfers", "filesize")
				tableShouldNotHaveColumns(t, eng.DB, "transfer_history", "filesize")
			})
		})
	})

	return mig
}

func testVer0_5_0TransferChangePaths(t *testing.T, eng *testEngine) Change {
	mig := Migrations[10]

	t.Run("When applying the 0.5.0 transfer paths changes", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO rules (
            id,name,send,comment,path,local_dir,remote_dir,tmp_local_receive_dir)
			VALUES (1,'snd',true,'','/snd_path','snd_loc','snd_rem','snd_tmp'),
			       (2,'rcv',false,'','/rcv_path','rcv_loc','rcv_rem','rcv_tmp')`)

		eng.NoError(t, `INSERT INTO transfers (id,owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,true_filepath,source_file,dest_file,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			(10,'', '', false,1,0,0,'/C:/src1','src1','dst1','1999-01-08 04:05:06 -8:00',
				'','',0,0,'',''),
			(20,'', '', false,2,0,0,'/C:/dst2','src2','dst2','1999-01-08 04:05:06 -8:00',
			 	'','',0,0,'','')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM transfers")
			eng.NoError(t, "DELETE FROM rules")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the paths", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "local_path")
			tableShouldNotHaveColumns(t, eng.DB, "transfers", "true_filepath")
		})

		t.Run("Then it should have replaced the source & dest file columns", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "remote_path")
			tableShouldNotHaveColumns(t, eng.DB, "transfers", "source_file", "dest_file")

			rows, err := eng.DB.Query(`SELECT transfers.remote_path, rules.send
				FROM transfers INNER JOIN rules ON transfers.rule_id=rules.id
				ORDER BY transfers.id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				path string
				send bool
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&path, &send))
			assert.True(t, send)
			assert.Equal(t, "snd_rem/dst1", path)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&path, &send))
			assert.False(t, send)
			assert.Equal(t, "rcv_rem/src2", path)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT transfers.true_filepath,
					transfers.source_file,transfers.dest_file,rules.send FROM
					transfers INNER JOIN rules ON transfers.rule_id=rules.id
					ORDER BY transfers.id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					full, src, dst string
					send           bool
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&full, &src, &dst, &send))
				assert.True(t, send)
				assert.Equal(t, "src1", src)
				assert.Equal(t, "dst1", dst)
				assert.Equal(t, "/C:/"+src, full)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&full, &src, &dst, &send))
				assert.False(t, send)
				assert.Equal(t, "src2", src)
				assert.Equal(t, "dst2", dst)
				assert.Equal(t, "/C:/"+dst, full)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_5_0TransferFormatLocalPath(t *testing.T, eng *testEngine) Change {
	mig := Migrations[11]

	t.Run("When applying the 0.5.0 transfer local path formatting", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO transfers (owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,local_path,remote_path,start,
			status,step,progression,task_number,error_code,error_details) VALUES
			('', '', false,1,0,0,'/C:/src','remote/dst','1999-01-08 04:05:06 -8:00',
			 '','',0,0,'','')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM transfers")
		})

		require.NoError(t,
			eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have formatted the local path to the OS format", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT local_path FROM transfers`)

			var path string
			require.NoError(t, row.Scan(&path))

			if isWindowsRuntime() {
				assert.Equal(t, `C:\src`, path)
			} else {
				assert.Equal(t, "/C:/src", path)
			}
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the local path to a URI", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT local_path FROM transfers`)

				var path string
				require.NoError(t, row.Scan(&path))
				assert.Equal(t, "/C:/src", path)
			})
		})
	})

	return mig
}

func testVer0_5_0HistoryChangePaths(t *testing.T, eng *testEngine) Change {
	mig := Migrations[12]

	t.Run("When applying the 0.5.0 history paths changes", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO transfer_history (id,owner,remote_transfer_id,
        	protocol,is_server,is_send,rule,agent,account,source_filename,dest_filename,
			start,stop,status,step,progression,task_number,error_code,error_details) VALUES
			(1,'','','',false,true,'','','','src_file','dst_file','1999-01-08 04:05:06 -8:00',
			 '1999-01-08 04:05:06 -8:00','','',0,0,'','')`)

		eng.NoError(t, `INSERT INTO transfer_history (id,owner,remote_transfer_id,
        	protocol,is_server,is_send,rule,agent,account,source_filename,dest_filename,
			start,stop,status,step,progression,task_number,error_code,error_details) VALUES
			(2,'','','',false,false,'','','','src_file','dst_file','1999-01-08 04:05:06 -8:00',
			 '1999-01-08 04:05:06 -8:00','','',0,0,'','')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM transfer_history")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have renamed the filename columns", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfer_history", "local_path", "remote_path")
			tableShouldNotHaveColumns(t, eng.DB, "transfers", "source_filename", "dest_filename")

			rows, err := eng.DB.Query(`SELECT is_send,local_path,remote_path 
				FROM transfer_history ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				send     bool
				loc, rem string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&send, &loc, &rem))
			assert.True(t, send)
			assert.Equal(t, "src_file", loc)
			assert.Equal(t, "dst_file", rem)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&send, &loc, &rem))
			assert.False(t, send)
			assert.Equal(t, "dst_file", loc)
			assert.Equal(t, "src_file", rem)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the changes", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfer_history", "local_path", "remote_path")
				tableShouldHaveColumns(t, eng.DB, "transfer_history", "source_filename", "dest_filename")

				rows, err := eng.DB.Query(`SELECT source_filename,dest_filename
					FROM transfer_history ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var src, dst string

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&src, &dst))
				assert.Equal(t, "src_file", src)
				assert.Equal(t, "dst_file", dst)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&src, &dst))
				assert.Equal(t, "src_file", src)
				assert.Equal(t, "dst_file", dst)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_5_0LocalAccountsPasswordDecode(t *testing.T, eng *testEngine) Change {
	mig := Migrations[13]

	t.Run("When applying the 0.5.0 local accounts password decoding", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO local_accounts (id,local_agent_id,
				login,password_hash) VALUES (1,'1','toto','Zm9vYmFy')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM local_accounts")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have decoded the password hash", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT password_hash FROM local_accounts`)

			var hash string
			require.NoError(t, row.Scan(&hash))
			assert.Equal(t, "foobar", hash)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have re-encoded the password hash", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT password_hash FROM local_accounts`)

				var hash string
				require.NoError(t, row.Scan(&hash))
				assert.Equal(t, "Zm9vYmFy", hash)
			})
		})
	})

	return mig
}

func testVer0_5_0UserPasswordChange(t *testing.T, eng *testEngine) Change {
	mig := Migrations[14]

	t.Run("When applying the 0.5.0 user password changes", func(t *testing.T) {
		pswd := []byte("Zm9vYmFy")
		perm := []byte{0b10101010}

		eng.NoError(t, `INSERT INTO users (owner,id,username,password,
		   permissions)	VALUES ('gw',1,'toto',?,?)`, pswd, perm)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM users")
		})

		require.NoError(t,
			eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the password column", func(t *testing.T) {
			tableShouldNotHaveColumns(t, eng.DB, "users", "password")
			tableShouldHaveColumns(t, eng.DB, "users", "password_hash")

			row := eng.DB.QueryRow(`SELECT password_hash FROM users`)

			var hash string
			require.NoError(t, row.Scan(&hash))
			assert.Equal(t, "Zm9vYmFy", hash)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t,
				eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the password changes", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "users", "password_hash")
				tableShouldHaveColumns(t, eng.DB, "users", "password")

				row := eng.DB.QueryRow(`SELECT password FROM users`)

				var hash []byte
				require.NoError(t, row.Scan(&hash))
				assert.Equal(t, []byte("Zm9vYmFy"), hash)
			})
		})
	})

	return mig
}
