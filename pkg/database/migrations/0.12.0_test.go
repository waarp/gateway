package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_12_0AddCryptoKeys(t *testing.T, eng *testEngine) Change {
	mig := Migrations[56]

	t.Run("When applying the 0.12.0 crypto keys addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))
			tableShouldHaveColumns(t, eng.DB, "crypto_keys",
				"id", "name", "type", "key")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_keys"))
			})
		})
	})

	return mig
}

func testVer0_12_0DropRemoteTransferIdUnique(t *testing.T, eng *testEngine) Change {
	mig := Migrations[57]

	t.Run("When applying the 0.12.0 remote transfer ID unique removal", func(t *testing.T) {
		// setup
		eng.NoError(t, `INSERT INTO 
    		rules (id, name, path, is_send) 
			VALUES(1, 'send', 'send', true)`,
		)
		t.Cleanup(func() { eng.NoError(t, `DELETE FROM rules`) })

		eng.NoError(t, `INSERT INTO 
    		local_agents(id, owner, name, protocol, address)
			VALUES(10, 'gw', 'serv', 'sftp', 'localhost:10')`,
		)
		t.Cleanup(func() { eng.NoError(t, `DELETE FROM local_agents`) })

		eng.NoError(t, `INSERT INTO 
    		local_accounts(id, local_agent_id, login)
			VALUES(100, 10, 'user')`,
		)
		t.Cleanup(func() { eng.NoError(t, `DELETE FROM local_accounts`) })

		// existing transfer
		eng.NoError(t, `INSERT INTO 
    		transfers(id, owner, remote_transfer_id, rule_id, local_account_id, src_filename)
			VALUES(1000, 'gw', '123', 1, 100, 'file.1')`,
		)
		t.Cleanup(func() { eng.NoError(t, `DELETE FROM transfers`) })

		// duplicate ID should fail
		require.Error(t, eng.Exec(`INSERT INTO 
    		transfers(id, owner, remote_transfer_id, rule_id, local_account_id, src_filename)
			VALUES   (1001, 'gw', '123', 1, 100, 'file.2')`,
		))

		// existing history
		eng.NoError(t, `INSERT INTO
    		transfer_history(id, owner, remote_transfer_id, protocol, is_send,
				is_server, rule, client, account, agent, start, status, step)
			VALUES(10000, 'gw', '456', 'sftp', true, false, 'send', '', 'user', 'serv',
				'2022-01-01 01:00:00', 'DONE', 'NONE')`,
		)
		t.Cleanup(func() { eng.NoError(t, `DELETE FROM transfer_history`) })

		// duplicate ID should fail
		require.Error(t, eng.Exec(`INSERT INTO
    		transfer_history(id, owner, remote_transfer_id, protocol, is_send,
				is_server, rule, client, account, agent, start, status, step)
			VALUES(10001, 'gw', '456', 'sftp', true, false, 'send', '', 'user', 'serv',
				'2022-01-01T02:00:00Z', 'DONE', 'NONE')`,
		))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have dropped the constraints", func(t *testing.T) {
			// duplicate ID should no longer fail
			eng.NoError(t, `INSERT INTO 
    			transfers(id, owner, remote_transfer_id, rule_id, local_account_id, src_filename)
				VALUES(1001, 'gw', '123', 1, 100, 'file.2')`,
			)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM transfers WHERE id=1001`) })

			eng.NoError(t, `INSERT INTO
    		transfer_history(id, owner, remote_transfer_id, protocol, is_send,
				is_server, rule, client, account, agent, start, status, step)
			VALUES(10001, 'gw', '456', 'sftp', true, false, 'send', '', 'user', 'serv',
				'2022-01-01 02:00:00', 'DONE', 'NONE')`,
			)
			t.Cleanup(func() { eng.NoError(t, `DELETE FROM transfer_history WHERE id=10001`) })
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the constraints", func(t *testing.T) {
				// duplicate ID should fail again
				require.Error(t, eng.Exec(`INSERT INTO 
    				transfers(id, owner, remote_transfer_id, rule_id, 
    				          local_account_id, src_filename)
					VALUES(1001, 'gw', '123', 1, 100, 'file.2')`,
				))

				require.Error(t, eng.Exec(`INSERT INTO
    				transfer_history(id, owner, remote_transfer_id, protocol, is_send,
						is_server, rule, client, account, agent, start, status, step)
					VALUES(10001, 'gw', '456', 'sftp', true, false, 'send', '', 'user', 'serv',
						'2022-01-01 02:00:00', 'DONE', 'NONE')`,
				))
			})
		})
	})

	return mig
}
