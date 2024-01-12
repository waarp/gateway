package migrations

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/bits"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_7_0AddLocalAgentEnabled(t *testing.T, eng *testEngine) Change {
	mig := Migrations[17]

	t.Run("When applying the 0.7.0 server 'enable' addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "local_agents", "enabled")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "local_agents", "enabled")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "local_agents", "enabled")
			})
		})
	})

	return mig
}

func testVer0_7_0RevampUsersTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[18]

	t.Run("When applying the 0.7.0 'users' table revamp", func(t *testing.T) {
		const permBefore uint32 = 0xA0000003
		permAfter := bits.Reverse32(permBefore)
		mask := make([]byte, 4)

		binary.LittleEndian.PutUint32(mask, permBefore)

		if eng.Dialect == PostgreSQL {
			_, err := eng.DB.Exec(`INSERT INTO users(id,owner,username,
            	password_hash,permissions) VALUES (1,'waarp_gw','toto',
				'pswd_hash',$1)`, mask)
			require.NoError(t, err)
		} else {
			_, err := eng.DB.Exec(`INSERT INTO users(id,owner,username,
				password_hash,permissions) VALUES (1,'waarp_gw','toto',
				'pswd_hash',?)`, mask)
			require.NoError(t, err)
		}

		t.Cleanup(func() {
			_, err := eng.DB.Exec(`DELETE FROM users`)
			require.NoError(t, err)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			doesIndexExist(t, eng.DB, eng.Dialect, "users", "unique_username")

			row := eng.DB.QueryRow(`SELECT id,owner,username,password_hash,
       				permissions FROM users`)

			var (
				id                int
				perm              int32
				owner, name, hash string
			)

			require.NoError(t, row.Scan(&id, &owner, &name, &hash, &perm))

			assert.Equal(t, 1, id)
			assert.Equal(t, "waarp_gw", owner)
			assert.Equal(t, "toto", name)
			assert.Equal(t, "pswd_hash", hash)
			assert.Equal(t,
				fmt.Sprintf("%#b", permAfter),
				fmt.Sprintf("%#b", uint32(perm)))
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,owner,username,password_hash,
       					permissions FROM users`)

				var (
					id                int
					owner, name, hash string
					perm              []byte
				)

				require.NoError(t, row.Scan(&id, &owner, &name, &hash, &perm))

				assert.Equal(t, 1, id)
				assert.Equal(t, "waarp_gw", owner)
				assert.Equal(t, "toto", name)
				assert.Equal(t, "pswd_hash", hash)
				assert.Equal(t, mask, perm)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampLocalAgentTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[19]

	t.Run("When applying the 0.7.0 'local_agents' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (1,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{"key":"val"}')`)
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT id,owner,name,protocol,address,root_dir,
       				receive_dir,send_dir,tmp_receive_dir,proto_config FROM local_agents`)

			var (
				id                                                    int
				owner, name, proto, addr, root, recv, send, tmp, conf string
			)

			require.NoError(t, row.Scan(&id, &owner, &name, &proto, &addr,
				&root, &recv, &send, &tmp, &conf))

			assert.Equal(t, 1, id)
			assert.Equal(t, "waarp_gw", owner)
			assert.Equal(t, "sftp_serv", name)
			assert.Equal(t, "sftp", proto)
			assert.Equal(t, "localhost:2222", addr)
			assert.Equal(t, "root", root)
			assert.Equal(t, "rcv", recv)
			assert.Equal(t, "snd", send)
			assert.Equal(t, "tmp", tmp)
			assert.Equal(t, `{"key":"val"}`, conf)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,owner,name,protocol,address,root_dir,
       				receive_dir,send_dir,tmp_receive_dir,proto_config FROM local_agents`)

				var (
					id                                              int
					owner, name, proto, addr, root, recv, send, tmp string
					conf                                            []byte
				)

				require.NoError(t, row.Scan(&id, &owner, &name, &proto, &addr,
					&root, &recv, &send, &tmp, &conf))

				assert.Equal(t, 1, id)
				assert.Equal(t, "waarp_gw", owner)
				assert.Equal(t, "sftp_serv", name)
				assert.Equal(t, "sftp", proto)
				assert.Equal(t, "localhost:2222", addr)
				assert.Equal(t, "root", root)
				assert.Equal(t, "rcv", recv)
				assert.Equal(t, "snd", send)
				assert.Equal(t, "tmp", tmp)
				assert.Equal(t, []byte(`{"key":"val"}`), conf)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampRemoteAgentTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[20]

	t.Run("When applying the 0.7.0 'remote_agents' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
			proto_config) VALUES (1,'sftp_part','sftp','localhost:2222','{}')`)
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT id,name,protocol,address,
       				proto_config FROM remote_agents`)

			var (
				id                      int
				name, proto, addr, conf string
			)

			require.NoError(t, row.Scan(&id, &name, &proto, &addr, &conf))

			assert.Equal(t, 1, id)
			assert.Equal(t, "sftp_part", name)
			assert.Equal(t, "sftp", proto)
			assert.Equal(t, "localhost:2222", addr)
			assert.Equal(t, "{}", conf)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,name,protocol,address,
       					proto_config FROM remote_agents`)

				var (
					id                int
					name, proto, addr string
					conf              []byte
				)

				require.NoError(t, row.Scan(&id, &name, &proto, &addr, &conf))

				assert.Equal(t, 1, id)
				assert.Equal(t, "sftp_part", name)
				assert.Equal(t, "sftp", proto)
				assert.Equal(t, "localhost:2222", addr)
				assert.Equal(t, []byte("{}"), conf)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampLocalAccountsTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[21]

	t.Run("When applying the 0.7.0 'local_accounts' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (1,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err1)

		if eng.Dialect == PostgreSQL {
			_, err2 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,
				login,password_hash) VALUES (10,1,'toto',$1)`, []byte("pswdhash"))
			require.NoError(t, err2)
		} else {
			_, err2 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,
				login,password_hash) VALUES (10,1,'toto',?)`, []byte("pswdhash"))
			require.NoError(t, err2)
		}

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err2)
			_, err3 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err3)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT id,local_agent_id,login,
       				password_hash FROM local_accounts`)

			var (
				id, agID    int
				login, hash string
			)

			require.NoError(t, row.Scan(&id, &agID, &login, &hash))

			assert.Equal(t, 10, id)
			assert.Equal(t, 1, agID)
			assert.Equal(t, "toto", login)
			assert.Equal(t, "pswdhash", hash)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,local_agent_id,login,
       					password_hash FROM local_accounts`)

				var (
					id, agID    int
					login, hash string
				)

				require.NoError(t, row.Scan(&id, &agID, &login, &hash))

				assert.Equal(t, 10, id)
				assert.Equal(t, 1, agID)
				assert.Equal(t, "toto", login)
				assert.Equal(t, "pswdhash", hash)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampRemoteAccountsTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[22]

	t.Run("When applying the 0.7.0 'remote_accounts' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (1,'sftp_part','sftp','localhost:2222','{}')`)
		require.NoError(t, err1)

		_, err2 := eng.DB.Exec(`INSERT INTO
    			remote_accounts(id,remote_agent_id,login,password)
				VALUES (10,1,'toto','pswd'), (20,1,'titi',NULL)`)
		require.NoError(t, err2)

		t.Cleanup(func() {
			_, err3 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err3)
			_, err4 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err4)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,remote_agent_id,login,
       				password FROM remote_accounts`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id, agID   int
				login, pwd string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &agID, &login, &pwd))
			assert.Equal(t, 10, id)
			assert.Equal(t, 1, agID)
			assert.Equal(t, "toto", login)
			assert.Equal(t, "pswd", pwd)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &agID, &login, &pwd))
			assert.Equal(t, 20, id)
			assert.Equal(t, 1, agID)
			assert.Equal(t, "titi", login)
			assert.Equal(t, "", pwd)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,remote_agent_id,login,
       					password FROM remote_accounts`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id, agID int
					login    string
					pwd      sql.NullString
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &agID, &login, &pwd))
				assert.Equal(t, 10, id)
				assert.Equal(t, 1, agID)
				assert.Equal(t, "toto", login)
				assert.True(t, pwd.Valid)
				assert.Equal(t, "pswd", pwd.String)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &agID, &login, &pwd))
				assert.Equal(t, 20, id)
				assert.Equal(t, 1, agID)
				assert.Equal(t, "titi", login)
				assert.False(t, pwd.Valid)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_7_0RevampRulesTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[23]

	t.Run("When applying the 0.7.0 'rules' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT id,name,is_send,comment,path,
            		local_dir,remote_dir,tmp_local_receive_dir FROM rules`)

			var (
				id                                 int
				name, comment, path, loc, rem, tmp string
				send                               bool
			)

			require.NoError(t, row.Scan(&id, &name, &send, &comment, &path,
				&loc, &rem, &tmp))

			assert.Equal(t, 1, id)
			assert.Equal(t, "push", name)
			assert.True(t, send)
			assert.Equal(t, "this is a comment", comment)
			assert.Equal(t, "/push", path)
			assert.Equal(t, "locDir", loc)
			assert.Equal(t, "remDir", rem)
			assert.Equal(t, "tmpDir", tmp)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,name,send,comment,path,
            		local_dir,remote_dir,tmp_local_receive_dir FROM rules`)

				var (
					id                                 int
					name, comment, path, loc, rem, tmp string
					send                               bool
				)

				require.NoError(t, row.Scan(&id, &name, &send, &comment, &path,
					&loc, &rem, &tmp))

				assert.Equal(t, 1, id)
				assert.Equal(t, "push", name)
				assert.True(t, send)
				assert.Equal(t, "this is a comment", comment)
				assert.Equal(t, "/push", path)
				assert.Equal(t, "locDir", loc)
				assert.Equal(t, "remDir", rem)
				assert.Equal(t, "tmpDir", tmp)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampTasksTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[24]

	t.Run("When applying the 0.7.0 'tasks' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		require.NoError(t, err1)

		if eng.Dialect == PostgreSQL {
			_, err2 := eng.DB.Exec(`INSERT INTO tasks(rule_id,chain,rank,type,args)
				VALUES (1,'POST',0,'DELETE',$1)`, []byte("{}"))
			require.NoError(t, err2)
		} else {
			_, err2 := eng.DB.Exec(`INSERT INTO tasks(rule_id,chain,rank,type,args)
				VALUES (1,'POST',0,'DELETE',?)`, []byte("{}"))
			require.NoError(t, err2)
		}

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM tasks`)
			require.NoError(t, err2)
			_, err3 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err3)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT rule_id,chain,rank,type,args FROM tasks`)

			var (
				rID, rank        int
				chain, typ, args string
			)

			require.NoError(t, row.Scan(&rID, &chain, &rank, &typ, &args))

			assert.Equal(t, 1, rID)
			assert.Equal(t, "POST", chain)
			assert.Equal(t, 0, rank)
			assert.Equal(t, "DELETE", typ)
			assert.Equal(t, "{}", args)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT rule_id,chain,rank,type,args FROM tasks`)

				var (
					rID, rank  int
					chain, typ string
					args       []byte
				)

				require.NoError(t, row.Scan(&rID, &chain, &rank, &typ, &args))

				assert.Equal(t, 1, rID)
				assert.Equal(t, "POST", chain)
				assert.Equal(t, 0, rank)
				assert.Equal(t, "DELETE", typ)
				assert.Equal(t, []byte("{}"), args)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampHistoryTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[25]

	t.Run("When applying the 0.7.0 'transfer_history' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            is_send,remote_transfer_id,rule,account,agent,protocol,local_path,
            remote_path,filesize,start,stop,status,step,progression,task_number,
            error_code,error_details) VALUES (1,'waarp_gw',FALSE,TRUE,'abc','push',
            'toto','sftp_part','sftp','/loc/path','/rem/path',123,'2021-01-01T01:00:00.123456Z',
            '2021-01-01T02:00:00.123456Z','CANCELLED','StepData',111,12,'TeDataTransfer',
            'this is an error message')`)
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM transfer_history`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			row := eng.DB.QueryRow(`SELECT id,owner,is_server,is_send,
       				remote_transfer_id,rule,account,agent,protocol,local_path,
            		remote_path,filesize,start,stop,status,step,progress,task_number,
           			error_code,error_details FROM transfer_history`)

			var (
				id, prog, size, taskNb                int
				serv, send                            bool
				start, stop                           time.Time
				owner, remID, rule, acc, ag, proto    string
				lPath, rPath, stat, step, eCode, eDet string
			)

			require.NoError(t, row.Scan(&id, &owner, &serv, &send, &remID, &rule,
				&acc, &ag, &proto, &lPath, &rPath, &size, &start, &stop, &stat,
				&step, &prog, &taskNb, &eCode, &eDet))

			assert.Equal(t, 1, id)
			assert.Equal(t, "waarp_gw", owner)
			assert.False(t, serv)
			assert.True(t, send)
			assert.Equal(t, "abc", remID)
			assert.Equal(t, "push", rule)
			assert.Equal(t, "toto", acc)
			assert.Equal(t, "sftp_part", ag)
			assert.Equal(t, "sftp", proto)
			assert.Equal(t, "/loc/path", lPath)
			assert.Equal(t, "/rem/path", rPath)
			assert.Equal(t, 123, size)
			assert.Equal(t, time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.UTC), start)
			assert.Equal(t, time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.UTC), stop)
			assert.Equal(t, "CANCELLED", stat) //nolint:misspell //must be kept for retro-compatibility
			assert.Equal(t, "StepData", step)
			assert.Equal(t, 111, prog)
			assert.Equal(t, 12, taskNb)
			assert.Equal(t, "TeDataTransfer", eCode)
			assert.Equal(t, "this is an error message", eDet)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT id,owner,is_server,is_send,
       				remote_transfer_id,rule,account,agent,protocol,local_path,
            		remote_path,filesize,start,stop,status,step,progression,
       				task_number,error_code,error_details FROM transfer_history`)

				var (
					id, prog, size, taskNb                int
					serv, send                            bool
					start, stop                           string
					owner, remID, rule, acc, ag, proto    string
					lPath, rPath, stat, step, eCode, eDet string
				)

				require.NoError(t, row.Scan(&id, &owner, &serv, &send, &remID,
					&rule, &acc, &ag, &proto, &lPath, &rPath, &size, &start,
					&stop, &stat, &step, &prog, &taskNb, &eCode, &eDet))

				startDate, startErr := time.Parse(time.RFC3339Nano, start)
				require.NoError(t, startErr)
				stopDate, stopErr := time.Parse(time.RFC3339Nano, stop)
				require.NoError(t, stopErr)

				assert.Equal(t, 1, id)
				assert.Equal(t, "waarp_gw", owner)
				assert.False(t, serv)
				assert.True(t, send)
				assert.Equal(t, "abc", remID)
				assert.Equal(t, "push", rule)
				assert.Equal(t, "toto", acc)
				assert.Equal(t, "sftp_part", ag)
				assert.Equal(t, "sftp", proto)
				assert.Equal(t, "/loc/path", lPath)
				assert.Equal(t, "/rem/path", rPath)
				assert.Equal(t, 123, size)
				assert.Equal(t,
					time.Date(2021, 1, 1, 1, 0, 0, 123456000, time.UTC),
					startDate.UTC())
				assert.Equal(t,
					time.Date(2021, 1, 1, 2, 0, 0, 123456000, time.UTC),
					stopDate.UTC())
				assert.Equal(t, "CANCELLED", stat) //nolint:misspell //must be kept for retro-compatibility
				assert.Equal(t, "StepData", step)
				assert.Equal(t, 111, prog)
				assert.Equal(t, 12, taskNb)
				assert.Equal(t, "TeDataTransfer", eCode)
				assert.Equal(t, "this is an error message", eDet)
			})
		})
	})

	return mig
}

func testVer0_7_0RevampTransfersTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[26]

	t.Run("When applying the 0.7.0 'transfers' table revamp", func(t *testing.T) {
		// ### Rule ###
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		require.NoError(t, err1)

		// ### Local ###
		_, err2 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err2)

		_, err3 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100,10,'toto')`)
		require.NoError(t, err3)

		_, err4 := eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,is_server,agent_id,account_id,local_path,remote_path,filesize,
            start,status,step,progression,task_number,error_code,error_details)
            VALUES (1000,'waarp_gw','abcd',1,TRUE,10,100,'/loc/path/1','/rem/path/1',
            1234,'2021-01-01T01:00:00Z','RUNNING','StepData',456,5,'TeDataTransfer',
            'this is an error message')`)
		require.NoError(t, err4)

		// ### Remote ###
		_, err5 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		require.NoError(t, err5)

		_, err6 := eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200,20,'toto')`)
		require.NoError(t, err6)

		_, err7 := eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,is_server,agent_id,account_id,local_path,remote_path,filesize,
            start,status,step,progression,task_number,error_code,error_details)
            VALUES (2000,'waarp_gw','efgh',1,FALSE,20,200,'/loc/path/2','/rem/path/2',
            5678,'2021-01-02T01:00:00Z','INTERRUPTED','StepPostTasks',789,8,
            'TeExternalOp','this is another error message')`)
		require.NoError(t, err7)

		t.Cleanup(func() {
			_, err8 := eng.DB.Exec(`DELETE FROM transfers`)
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err10)
			_, err11 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err11)
			_, err12 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err12)
			_, err13 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err13)
			_, err14 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err14)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,owner,remote_transfer_id,rule_id,
				local_account_id,remote_account_id,local_path,remote_path,filesize,
				start,status,step,progress,task_number,error_code,error_details
				FROM transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id, ruleID, size, prog, taskNb                        int
				lAcc, rAcc                                            sql.NullInt64
				owner, remID, lPath, rPath, status, step, eCode, eDet string
				start                                                 time.Time
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remID, &ruleID, &lAcc,
				&rAcc, &lPath, &rPath, &size, &start, &status, &step, &prog,
				&taskNb, &eCode, &eDet))

			assert.Equal(t, 1000, id)
			assert.Equal(t, "waarp_gw", owner)
			assert.Equal(t, "abcd", remID)
			assert.Equal(t, 1, ruleID)
			assert.Equal(t, int64(100), lAcc.Int64)
			assert.False(t, rAcc.Valid)
			assert.Equal(t, "/loc/path/1", lPath)
			assert.Equal(t, "/rem/path/1", rPath)
			assert.Equal(t, 1234, size)
			assert.Equal(t, time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), start)
			assert.Equal(t, "RUNNING", status)
			assert.Equal(t, "StepData", step)
			assert.Equal(t, 456, prog)
			assert.Equal(t, 5, taskNb)
			assert.Equal(t, "TeDataTransfer", eCode)
			assert.Equal(t, "this is an error message", eDet)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remID, &ruleID, &lAcc,
				&rAcc, &lPath, &rPath, &size, &start, &status, &step, &prog,
				&taskNb, &eCode, &eDet))

			assert.Equal(t, 2000, id)
			assert.Equal(t, "waarp_gw", owner)
			assert.Equal(t, "efgh", remID)
			assert.Equal(t, 1, ruleID)
			assert.False(t, lAcc.Valid)
			assert.Equal(t, int64(200), rAcc.Int64)
			assert.Equal(t, "/loc/path/2", lPath)
			assert.Equal(t, "/rem/path/2", rPath)
			assert.Equal(t, 5678, size)
			assert.Equal(t, time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC), start)
			assert.Equal(t, "INTERRUPTED", status)
			assert.Equal(t, "StepPostTasks", step)
			assert.Equal(t, 789, prog)
			assert.Equal(t, 8, taskNb)
			assert.Equal(t, "TeExternalOp", eCode)
			assert.Equal(t, "this is another error message", eDet)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,remote_transfer_id,
					owner,rule_id,is_server,agent_id,account_id,local_path,
					remote_path,filesize,start,status,step,progression,
					task_number,error_code,error_details FROM transfers
					ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id, ruleID, agID, accID, size, prog, taskNb                  int
					isServ                                                       bool
					remID, owner, lPath, rPath, start, status, step, eCode, eDet string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &remID, &owner, &ruleID,
					&isServ, &agID, &accID, &lPath, &rPath, &size, &start,
					&status, &step, &prog, &taskNb, &eCode, &eDet))

				startDate, startErr := time.Parse(time.RFC3339Nano, start)
				require.NoError(t, startErr)

				assert.Equal(t, 1000, id)
				assert.Equal(t, "waarp_gw", owner)
				assert.Equal(t, "abcd", remID)
				assert.Equal(t, 1, ruleID)
				assert.True(t, isServ)
				assert.Equal(t, 10, agID)
				assert.Equal(t, 100, accID)
				assert.Equal(t, "/loc/path/1", lPath)
				assert.Equal(t, "/rem/path/1", rPath)
				assert.Equal(t, 1234, size)
				assert.Equal(t,
					time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
					startDate.UTC())
				assert.Equal(t, "RUNNING", status)
				assert.Equal(t, "StepData", step)
				assert.Equal(t, 456, prog)
				assert.Equal(t, 5, taskNb)
				assert.Equal(t, "TeDataTransfer", eCode)
				assert.Equal(t, "this is an error message", eDet)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &remID, &owner, &ruleID,
					&isServ, &agID, &accID, &lPath, &rPath, &size, &start,
					&status, &step, &prog, &taskNb, &eCode, &eDet))

				startDate, startErr = time.Parse(time.RFC3339Nano, start)
				require.NoError(t, startErr)

				assert.Equal(t, 2000, id)
				assert.Equal(t, "waarp_gw", owner)
				assert.Equal(t, "efgh", remID)
				assert.Equal(t, 1, ruleID)
				assert.False(t, isServ)
				assert.Equal(t, 20, agID)
				assert.Equal(t, 200, accID)
				assert.Equal(t, "/loc/path/2", lPath)
				assert.Equal(t, "/rem/path/2", rPath)
				assert.Equal(t, 5678, size)
				assert.Equal(t,
					time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC),
					startDate.UTC())
				assert.Equal(t, "INTERRUPTED", status)
				assert.Equal(t, "StepPostTasks", step)
				assert.Equal(t, 789, prog)
				assert.Equal(t, 8, taskNb)
				assert.Equal(t, "TeExternalOp", eCode)
				assert.Equal(t, "this is another error message", eDet)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_7_0RevampTransferInfoTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[27]

	t.Run("When applying the 0.7.0 'transfer_info' table revamp", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,comment,path,
            local_dir,remote_dir,tmp_local_receive_dir) VALUES (1,'push',TRUE,
            'this is a comment','/push','locDir','remDir','tmpDir')`)
		require.NoError(t, err1)

		_, err2 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err2)

		_, err3 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100,10,'toto')`)
		require.NoError(t, err3)

		_, err4 := eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,local_account_id,remote_account_id,local_path,remote_path,
            filesize,start,status,step,progress,task_number,error_code,error_details)
            VALUES (1000,'waarp_gw','abcd',1,100,null,'/loc/path/1','/rem/path/1',
            1234,'2021-01-01 01:00:00','RUNNING','StepData',456,5,'TeDataTransfer',
            'this is an error message')`)
		require.NoError(t, err4)

		_, err5 := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            is_send,remote_transfer_id,rule,account,agent,protocol,local_path,
            remote_path,filesize,start,stop,status,step,progress,task_number,
            error_code,error_details) VALUES (2000,'waarp_gw',FALSE,TRUE,'abc','push',
            'toto','sftp_part','sftp','/loc/path','/rem/path',123,'2021-01-01 01:00:00',
            '2021-01-01 02:00:00','CANCELLED','StepData',111,0,'TeDataTransfer',
            'this is an error message')`)
		require.NoError(t, err5)

		_, err6 := eng.DB.Exec(`INSERT INTO transfer_info(transfer_id,is_history,
        	name,value) VALUES (1000,false,'1_t_info','"t_value"'),
        	                   (2000,true, '2_h_info','"h_value"'),
        	                   (3000,true, '3_h_info_orphan','true')`) // orphaned entry
		require.NoError(t, err6)

		t.Cleanup(func() {
			_, err7 := eng.DB.Exec(`DELETE FROM transfer_info`)
			require.NoError(t, err7)
			_, err8 := eng.DB.Exec(`DELETE FROM transfer_history`)
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec(`DELETE FROM transfers`)
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err10)
			_, err11 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err11)
			_, err12 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err12)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT transfer_id,history_id,name,value
					FROM transfer_info ORDER BY name`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				tID, hID    sql.NullInt64
				name, value string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&tID, &hID, &name, &value))
			assert.Equal(t, int64(1000), tID.Int64)
			assert.False(t, hID.Valid)
			assert.Equal(t, "1_t_info", name)
			assert.Equal(t, `"t_value"`, value)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&tID, &hID, &name, &value))
			assert.False(t, tID.Valid)
			assert.Equal(t, int64(2000), hID.Int64)
			assert.Equal(t, "2_h_info", name)
			assert.Equal(t, `"h_value"`, value)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT transfer_id,is_history,name,value
					FROM transfer_info ORDER BY transfer_id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					tID         int
					isHist      bool
					name, value string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&tID, &isHist, &name, &value))
				assert.Equal(t, 1000, tID)
				assert.False(t, isHist)
				assert.Equal(t, "1_t_info", name)
				assert.Equal(t, `"t_value"`, value)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&tID, &isHist, &name, &value))
				assert.Equal(t, 2000, tID)
				assert.True(t, isHist)
				assert.Equal(t, "2_h_info", name)
				assert.Equal(t, `"h_value"`, value)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_7_0RevampCryptoTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[28]

	t.Run("When applying the 0.7.0 'crypto' table revamp", func(t *testing.T) {
		// ### Local agent ###
		_, err1 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err1)

		// ### Local account ###
		_, err2 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100,10,'toto')`)
		require.NoError(t, err2)

		// ### Remote agent ###
		_, err3 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		require.NoError(t, err3)

		// ### Remote account ###
		_, err4 := eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200,20,'toto')`)
		require.NoError(t, err4)

		// ##### Certificates #####
		_, err5 := eng.DB.Exec(`INSERT INTO crypto_credentials(id,name,owner_type,
            owner_id,private_key,certificate,ssh_public_key) VALUES 
            (1010,'lag_cert','local_agents',   10, 'pk1','cert1','pbk1'),
            (1100,'lac_cert','local_accounts', 100,'pk2','cert2','pbk2'),
            (2020,'rag_cert','remote_agents',  20, 'pk3','cert3','pbk3'),
            (2200,'rac_cert','remote_accounts',200, NULL,'cert4','pbk4')`)
		require.NoError(t, err5)

		t.Cleanup(func() {
			_, err6 := eng.DB.Exec(`DELETE FROM crypto_credentials`)
			require.NoError(t, err6)
			_, err7 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err7)
			_, err8 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err10)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,name,local_agent_id,
				remote_agent_id,local_account_id,remote_account_id,
				private_key,certificate,ssh_public_key FROM crypto_credentials
				ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                         int
				lagID, lacID, ragID, racID sql.NullInt64
				name, pk, cert, pbk        string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &name, &lagID, &ragID, &lacID,
				&racID, &pk, &cert, &pbk))
			assert.Equal(t, 1010, id)
			assert.Equal(t, "lag_cert", name)
			assert.Equal(t, int64(10), lagID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(0), racID.Int64)
			assert.Equal(t, "pk1", pk)
			assert.Equal(t, "cert1", cert)
			assert.Equal(t, "pbk1", pbk)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &name, &lagID, &ragID, &lacID,
				&racID, &pk, &cert, &pbk))
			assert.Equal(t, 1100, id)
			assert.Equal(t, "lac_cert", name)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(100), lacID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(0), racID.Int64)
			assert.Equal(t, "pk2", pk)
			assert.Equal(t, "cert2", cert)
			assert.Equal(t, "pbk2", pbk)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &name, &lagID, &ragID, &lacID,
				&racID, &pk, &cert, &pbk))
			assert.Equal(t, 2020, id)
			assert.Equal(t, "rag_cert", name)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(20), ragID.Int64)
			assert.Equal(t, int64(0), racID.Int64)
			assert.Equal(t, "pk3", pk)
			assert.Equal(t, "cert3", cert)
			assert.Equal(t, "pbk3", pbk)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &name, &lagID, &ragID, &lacID,
				&racID, &pk, &cert, &pbk))
			assert.Equal(t, 2200, id)
			assert.Equal(t, "rac_cert", name)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(200), racID.Int64)
			assert.Equal(t, "", pk)
			assert.Equal(t, "cert4", cert)
			assert.Equal(t, "pbk4", pbk)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,name,owner_type,owner_id,
       					private_key,certificate,ssh_public_key FROM crypto_credentials`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id, oID                int
					name, oType, cert, pbk string
					pk                     sql.NullString
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &oType, &oID, &pk, &cert, &pbk))
				assert.Equal(t, 1010, id)
				assert.Equal(t, "lag_cert", name)
				assert.Equal(t, "local_agents", oType)
				assert.Equal(t, 10, oID)
				assert.True(t, pk.Valid)
				assert.Equal(t, "pk1", pk.String)
				assert.Equal(t, "cert1", cert)
				assert.Equal(t, "pbk1", pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &oType, &oID, &pk, &cert, &pbk))
				assert.Equal(t, 1100, id)
				assert.Equal(t, "lac_cert", name)
				assert.Equal(t, "local_accounts", oType)
				assert.Equal(t, 100, oID)
				assert.True(t, pk.Valid)
				assert.Equal(t, "pk2", pk.String)
				assert.Equal(t, "cert2", cert)
				assert.Equal(t, "pbk2", pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &oType, &oID, &pk, &cert, &pbk))
				assert.Equal(t, 2020, id)
				assert.Equal(t, "rag_cert", name)
				assert.Equal(t, "remote_agents", oType)
				assert.Equal(t, 20, oID)
				assert.True(t, pk.Valid)
				assert.Equal(t, "pk3", pk.String)
				assert.Equal(t, "cert3", cert)
				assert.Equal(t, "pbk3", pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &name, &oType, &oID, &pk, &cert, &pbk))
				assert.Equal(t, 2200, id)
				assert.Equal(t, "rac_cert", name)
				assert.Equal(t, "remote_accounts", oType)
				assert.Equal(t, 200, oID)
				assert.False(t, pk.Valid)
				assert.Equal(t, "cert4", cert)
				assert.Equal(t, "pbk4", pbk)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_7_0RevampRuleAccessTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[29]

	t.Run("When applying the 0.7.0 'rule_access' table revamp", func(t *testing.T) {
		// ### Rule ###
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,path) 
			VALUES (1,'push1',TRUE,'/push1'), (2,'push2',TRUE,'/push2'),
			       (3,'push3',TRUE,'/push3'), (4,'push4',TRUE,'/push4')`)
		require.NoError(t, err1)

		// ### Local agent ###
		_, err2 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err2)

		// ### Local account ###
		_, err3 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100,10,'toto')`)
		require.NoError(t, err3)

		// ### Remote agent ###
		_, err4 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		require.NoError(t, err4)

		// ### Remote account ###
		_, err5 := eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200,20,'toto')`)
		require.NoError(t, err5)

		// ##### Accesses #####
		_, err6 := eng.DB.Exec(`INSERT INTO rule_access(rule_id,object_type,object_id)
			VALUES (1,'local_agents',  10),  (2,'remote_agents',  20),
			       (3,'local_accounts',100), (4,'remote_accounts',200)`)
		require.NoError(t, err6)

		t.Cleanup(func() {
			_, err7 := eng.DB.Exec("DELETE FROM rule_access")
			require.NoError(t, err7)
			_, err8 := eng.DB.Exec("DELETE FROM remote_accounts")
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec("DELETE FROM local_accounts")
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec("DELETE FROM remote_agents")
			require.NoError(t, err10)
			_, err11 := eng.DB.Exec("DELETE FROM local_agents")
			require.NoError(t, err11)
			_, err12 := eng.DB.Exec("DELETE FROM rules")
			require.NoError(t, err12)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have changed the columns", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT rule_id,local_agent_id,
       			remote_agent_id,local_account_id,remote_account_id 
				FROM rule_access ORDER BY rule_id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				ruleID                     int
				lagID, lacID, ragID, racID sql.NullInt64
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&ruleID, &lagID, &ragID, &lacID, &racID))
			assert.Equal(t, 1, ruleID)
			assert.Equal(t, int64(10), lagID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(0), racID.Int64)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&ruleID, &lagID, &ragID, &lacID, &racID))
			assert.Equal(t, 2, ruleID)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(20), ragID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(0), racID.Int64)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&ruleID, &lagID, &ragID, &lacID, &racID))
			assert.Equal(t, 3, ruleID)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(100), lacID.Int64)
			assert.Equal(t, int64(0), racID.Int64)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&ruleID, &lagID, &ragID, &lacID, &racID))
			assert.Equal(t, 4, ruleID)
			assert.Equal(t, int64(0), lagID.Int64)
			assert.Equal(t, int64(0), ragID.Int64)
			assert.Equal(t, int64(0), lacID.Int64)
			assert.Equal(t, int64(200), racID.Int64)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reverted the column changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT rule_id,object_type,
       					object_id FROM rule_access ORDER BY object_id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					ruleID, oID int
					oType       string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&ruleID, &oType, &oID))
				assert.Equal(t, 1, ruleID)
				assert.Equal(t, "local_agents", oType)
				assert.Equal(t, 10, oID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&ruleID, &oType, &oID))
				assert.Equal(t, 2, ruleID)
				assert.Equal(t, "remote_agents", oType)
				assert.Equal(t, 20, oID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&ruleID, &oType, &oID))
				assert.Equal(t, 3, ruleID)
				assert.Equal(t, "local_accounts", oType)
				assert.Equal(t, 100, oID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&ruleID, &oType, &oID))
				assert.Equal(t, 4, ruleID)
				assert.Equal(t, "remote_accounts", oType)
				assert.Equal(t, 200, oID)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_7_0AddLocalAgentsAddressUnique(t *testing.T, eng *testEngine) Change {
	mig := Migrations[30]

	t.Run("When applying the 0.7.0 transfers 'address' unique addition", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,address)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222')`)
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the unique constraint", func(t *testing.T) {
			_, err := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,address)
					VALUES (11,'waarp_gw','sftp_serv_2','sftp','localhost:2222')`)
			shouldBeUniqueViolationError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have removed the unique constraint", func(t *testing.T) {
				_, err := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,address)
						VALUES (11,'waarp_gw','sftp_serv_2','sftp','localhost:2222')`)
				require.NoError(t, err)
			})
		})
	})

	return mig
}

func testVer0_7_0AddNormalizedTransfersView(t *testing.T, eng *testEngine) Change {
	mig := Migrations[31]

	t.Run("When applying the 0.7.0 normalized transfer view addition", func(t *testing.T) {
		// ### Rules ###
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id,name,is_send,path)
			VALUES (1,'push',TRUE,'/push'), (2,'pull',FALSE,'/pull')`)
		require.NoError(t, err1)

		// ### Local ###
		_, err2 := eng.DB.Exec(`INSERT INTO local_agents(id,owner,name,protocol,
			address,root_dir,receive_dir,send_dir,tmp_receive_dir,proto_config)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222','root',
			        'rcv','snd','tmp','{}')`)
		require.NoError(t, err2)

		_, err3 := eng.DB.Exec(`INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100,10,'toto')`)
		require.NoError(t, err3)

		// ### Remote ###
		_, err4 := eng.DB.Exec(`INSERT INTO remote_agents(id,name,protocol,address,
        	proto_config) VALUES (20,'sftp_part','sftp','localhost:3333','{}')`)
		require.NoError(t, err4)

		_, err5 := eng.DB.Exec(`INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200,20,'tata')`)
		require.NoError(t, err5)

		// ### Transfers ###

		_, err6 := eng.DB.Exec(`INSERT INTO transfers(id,owner,remote_transfer_id,
            rule_id,local_account_id,remote_account_id,local_path,remote_path)
            VALUES (1000,'waarp_gw','efgh',1,100, null,'/loc/path/1','/rem/path/1'),
                   (2000,'waarp_gw','efgh',2,null,200, '/loc/path/2','/rem/path/2')`)
		require.NoError(t, err6)

		_, err7 := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,is_server,
            	is_send,remote_transfer_id,rule,account,agent,protocol,local_path,
            	remote_path,filesize,start,stop,status,step)
			VALUES (3000,'waarp_gw',FALSE,TRUE,'xyz','push','tutu','r66_part',
				'r66','/loc/path/3','/rem/path/3',123,'2021-01-03 01:00:00',
            	'2021-01-03 02:00:00','CANCELLED','StepData')`)
		require.NoError(t, err7)

		t.Cleanup(func() {
			_, err8 := eng.DB.Exec(`DELETE FROM transfer_history`)
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec(`DELETE FROM transfers`)
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err10)
			_, err11 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err11)
			_, err12 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err12)
			_, err13 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err13)
			_, err14 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err14)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the view", func(t *testing.T) {
			type normTrans struct {
				id                          int
				isServ, isSend, isTrans     bool
				rule, account, agent, proto string
			}

			rows, err := eng.DB.Query(`SELECT id,is_server,is_send,is_transfer,
       				rule,account,agent,protocol FROM normalized_transfers ORDER BY id`)
			require.NoError(t, err)
			defer rows.Close()

			var trans normTrans

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&trans.id, &trans.isServ, &trans.isSend,
				&trans.isTrans, &trans.rule, &trans.account, &trans.agent, &trans.proto))
			assert.Equal(t,
				normTrans{
					id: 1000, isServ: true, isSend: true, isTrans: true,
					rule: "push", account: "toto", agent: "sftp_serv", proto: "sftp",
				},
				trans)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&trans.id, &trans.isServ, &trans.isSend,
				&trans.isTrans, &trans.rule, &trans.account, &trans.agent, &trans.proto))
			assert.Equal(t,
				normTrans{
					id: 2000, isServ: false, isSend: false, isTrans: true,
					rule: "pull", account: "tata", agent: "sftp_part", proto: "sftp",
				},
				trans)
			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&trans.id, &trans.isServ, &trans.isSend,
				&trans.isTrans, &trans.rule, &trans.account, &trans.agent, &trans.proto))
			assert.Equal(t,
				normTrans{
					id: 3000, isServ: false, isSend: true, isTrans: false,
					rule: "push", account: "tutu", agent: "r66_part", proto: "r66",
				},
				trans)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the view", func(t *testing.T) {
				_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
				shouldBeTableNotExist(t, err)
			})
		})
	})

	return mig
}
