package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_8_0DropNormalizedTransfersView(t *testing.T, eng *testEngine) Change {
	mig := Migrations[33]

	t.Run("When applying the 0.8.0 normalized transfer view deletion", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
		require.NoError(t, err1)

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have dropped the view", func(t *testing.T) {
			_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
			shouldBeTableNotExist(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the view", func(t *testing.T) {
				_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
				require.NoError(t, err)
			})
		})
	})

	return mig
}

func testVer0_8_0AddTransferFilename(t *testing.T, eng *testEngine) Change {
	mig := Migrations[34]

	t.Run("When applying the 0.8.0 transfer filename addition", func(t *testing.T) {
		// ### Rule ###
		_, err1 := eng.DB.Exec(`INSERT INTO rules(id, name, path, is_send)
			VALUES (1, 'send', '/send', true), (2, 'recv', '/recv', false)`)
		require.NoError(t, err1)

		// ### Remote ###
		_, err2 := eng.DB.Exec(`INSERT INTO remote_agents(id, name, protocol, address)
			VALUES (10, 'sftp_part', 'sftp', '1.1.1.1:1111')`)
		require.NoError(t, err2)

		_, err3 := eng.DB.Exec(`INSERT INTO remote_accounts(id, remote_agent_id, login)
			VALUES (100, 10, 'toto')`)
		require.NoError(t, err3)

		// ### Local ###
		_, err4 := eng.DB.Exec(`INSERT INTO local_agents(id, owner, name, protocol, address)
			VALUES (20, 'waarp_gw', 'sftp_serv', 'sftp', 'localhost:2222')`)
		require.NoError(t, err4)

		_, err5 := eng.DB.Exec(`INSERT INTO local_accounts(id, local_agent_id, login)
			VALUES (200, 20, 'tata')`)
		require.NoError(t, err5)

		_, err6 := eng.DB.Exec(`INSERT INTO transfers(id, owner, remote_transfer_id,
            rule_id, local_account_id, remote_account_id, local_path, remote_path)
            VALUES (1000, 'waarp_gw', 'push', 1, null, 100, '/loc/path/1', '/rem/path/1'),
                   (2000, 'waarp_gw', 'pull', 2, null, 100, '/loc/path/2', '/rem/path/2'),
                   (3000, 'waarp_gw', 'push', 2, 200, null, '/loc/path/3', '/rem/path/3'),
                   (4000, 'waarp_gw', 'pull', 1, 200, null, '/loc/path/4', '/rem/path/4')`)
		require.NoError(t, err6)

		t.Cleanup(func() {
			_, err7 := eng.DB.Exec(`DELETE FROM transfers`)
			require.NoError(t, err7)
			_, err8 := eng.DB.Exec(`DELETE FROM local_accounts`)
			require.NoError(t, err8)
			_, err9 := eng.DB.Exec(`DELETE FROM remote_accounts`)
			require.NoError(t, err9)
			_, err10 := eng.DB.Exec(`DELETE FROM local_agents`)
			require.NoError(t, err10)
			_, err11 := eng.DB.Exec(`DELETE FROM remote_agents`)
			require.NoError(t, err11)
			_, err12 := eng.DB.Exec(`DELETE FROM rules`)
			require.NoError(t, err12)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added and filled the new column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "src_filename", "dest_filename")

			rows, err := eng.DB.Query(`SELECT id, src_filename, dest_filename,
				local_path, remote_path	FROM transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                                 int
				srcFile, dstFile, locPath, remPath string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 1000, id)
			assert.Equal(t, "/loc/path/1", srcFile)
			assert.Equal(t, "/rem/path/1", dstFile)
			assert.Equal(t, "/loc/path/1", locPath)
			assert.Equal(t, "/rem/path/1", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 2000, id)
			assert.Equal(t, "/rem/path/2", srcFile)
			assert.Equal(t, "/loc/path/2", dstFile)
			assert.Equal(t, "/loc/path/2", locPath)
			assert.Equal(t, "/rem/path/2", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 3000, id)
			assert.Equal(t, "", srcFile)
			assert.Equal(t, "/loc/path/3", dstFile)
			assert.Equal(t, "/loc/path/3", locPath)
			assert.Equal(t, "", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 4000, id)
			assert.Equal(t, "/loc/path/4", srcFile)
			assert.Equal(t, "", dstFile)
			assert.Equal(t, "/loc/path/4", locPath)
			assert.Equal(t, "", remPath)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("Then the local/remote path columns should no longer be mandatory", func(t *testing.T) {
			_, err := eng.DB.Exec(`INSERT INTO transfers(id, owner,
                    remote_transfer_id, rule_id, remote_account_id, src_filename)
            		VALUES (5000, 'waarp_gw', 'new', 1, 100, 'file5')`)
			require.NoError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the filename column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfers", "src_filename",
					"dest_filename")
			})

			t.Run("Then the local/remote path columns should again be mandatory", func(t *testing.T) {
				_, err := eng.DB.Exec(`INSERT INTO transfers(id, owner,
                    	remote_transfer_id, rule_id, remote_account_id)
            		VALUES (3000, 'waarp_gw', 'new', 1, 100)`)
				require.Error(t, err)
			})

			t.Run("Then it should have restored the remote path column for server transfers", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT remote_path FROM transfers
					WHERE local_account_id IS NOT NULL ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var remPath string

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remPath))
				assert.Equal(t, "/loc/path/3", remPath)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remPath))
				assert.Equal(t, "/loc/path/4", remPath)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_8_0AddHistoryFilename(t *testing.T, eng *testEngine) Change {
	mig := Migrations[35]

	t.Run("When applying the 0.8.0 history filename addition", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,remote_transfer_id,
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
		require.NoError(t, err1)

		t.Cleanup(func() {
			_, err2 := eng.DB.Exec(`DELETE FROM transfer_history`)
			require.NoError(t, err2)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added and filled the new column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfer_history",
				"src_filename", "dest_filename")

			rows, err := eng.DB.Query(`SELECT id, src_filename, dest_filename,
				local_path, remote_path	FROM transfer_history ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                                 int
				srcFile, dstFile, locPath, remPath string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 1, id)
			assert.Equal(t, "/loc/path/1", srcFile)
			assert.Equal(t, "/rem/path/1", dstFile)
			assert.Equal(t, "/loc/path/1", locPath)
			assert.Equal(t, "/rem/path/1", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 2, id)
			assert.Equal(t, "/rem/path/2", srcFile)
			assert.Equal(t, "/loc/path/2", dstFile)
			assert.Equal(t, "/loc/path/2", locPath)
			assert.Equal(t, "/rem/path/2", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 3, id)
			assert.Equal(t, "", srcFile)
			assert.Equal(t, "/loc/path/3", dstFile)
			assert.Equal(t, "/loc/path/3", locPath)
			assert.Equal(t, "", remPath)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &srcFile, &dstFile, &locPath, &remPath))
			assert.Equal(t, 4, id)
			assert.Equal(t, "/loc/path/4", srcFile)
			assert.Equal(t, "", dstFile)
			assert.Equal(t, "/loc/path/4", locPath)
			assert.Equal(t, "", remPath)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("Then the local/remote path columns should no longer be mandatory", func(t *testing.T) {
			_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,
					remote_transfer_id,is_server,is_send,account,agent,protocol,
                    src_filename,rule,start,status,step)
            		VALUES (5,'waarp_gw','555',false,true,'acc','ag','proto','file5',
							'push','2021-01-01 02:00:00.123456','DONE','StepNone')`)
			require.NoError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the filename column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfer_history",
					"src_filename", "dest_filename")
			})

			t.Run("Then the local/remote path columns should again be mandatory", func(t *testing.T) {
				_, err := eng.DB.Exec(`INSERT INTO transfer_history(id,owner,
					remote_transfer_id,is_server,is_send,account,agent,protocol,
                    rule,start,status,step)
            		VALUES (5,'waarp_gw','555',false,true,'acc','ag','proto','push',
            		        '2021-01-01 02:00:00.123456','DONE','StepNone')`)
				require.Error(t, err)
			})

			t.Run("Then it should have restored the remote path column for server transfers", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT remote_path FROM transfer_history
                	WHERE is_server=true ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var remPath string

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remPath))
				assert.Equal(t, "/loc/path/3", remPath)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remPath))
				assert.Equal(t, "/loc/path/4", remPath)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_8_0UpdateNormalizedTransfersView(t *testing.T, eng *testEngine) Change {
	mig := Migrations[36]

	t.Run("When applying the 0.8.0 normalized transfer view restoration", func(t *testing.T) {
		_, err1 := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
		require.Error(t, err1)

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have restored the view", func(t *testing.T) {
			_, err := eng.DB.Exec(`SELECT src_filename, dest_filename
				FROM normalized_transfers`)
			require.NoError(t, err)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the view", func(t *testing.T) {
				_, err := eng.DB.Exec(`SELECT * FROM normalized_transfers`)
				require.Error(t, err)
			})
		})
	})

	return mig
}
