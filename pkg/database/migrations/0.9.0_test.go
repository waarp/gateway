package migrations

import (
	"database/sql"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

func testVer0_9_0AddCloudInstances(t *testing.T, eng *testEngine) Change {
	mig := Migrations[37]

	t.Run("When applying the 0.9.0 cloud instances addition", func(t *testing.T) {
		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "cloud_instances"))

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have added the new table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "cloud_instances"))
			tableShouldHaveColumns(t, eng.DB, "cloud_instances", "id", "owner",
				"name", "type", "api_"+
					"key", "secret", "options")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the new table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "cloud_instances"))
			})
		})
	})

	return mig
}

func testVer0_9_0LocalPathToURL(t *testing.T, eng *testEngine) Change {
	mig := Migrations[38]

	osPath1, osPath2 := "/foo/bar/file.1", "/foo/bar/file.2"
	expURL1, expURL2 := "file:/foo/bar/file.1", "file:/foo/bar/file.2"

	if runtime.GOOS == "windows" {
		osPath1, osPath2 = `C:\foo\bar\file.1`, `C:\foo\bar\file.2`
		expURL1, expURL2 = `file:/C:/foo/bar/file.1`, `file:/C:/foo/bar/file.2`
	}

	t.Run("When applying the 0.9.0 local path to URL migration", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO rules 
    		(id, name, path, is_send) VALUES (1, 'recv', '/recv', false)`)

		eng.NoError(t, `INSERT INTO local_agents
    		(id, owner, name, protocol, address)
    		VALUES (10, 'gw', 'serv', 'sftp', 'localhost:10')`)

		eng.NoError(t, `INSERT INTO local_accounts
    		(id, local_agent_id, login) VALUES (100, 10, 'user')`)

		eng.NoError(t, `INSERT INTO transfers
			(id, owner, remote_transfer_id, rule_id, local_account_id, src_filename, local_path)
			VALUES (1000, 'gw', '1000', 1, 100, 'file.1', ?),
				(2000, 'gw', '2000', 1, 100, 'file.2', ?)`,
			osPath1, osPath2)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfers`)
			eng.NoError(t, `DELETE FROM local_accounts`)
			eng.NoError(t, `DELETE FROM local_agents`)
			eng.NoError(t, `DELETE FROM rules`)
		})

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have converted the local paths to URLs", func(t *testing.T) {
			rows, queryErr := eng.DB.Query(`SELECT id, local_path FROM transfers
					ORDER BY id`)
			require.NoError(t, queryErr)

			defer rows.Close()

			var (
				id   int
				path string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &path))
			assert.Equal(t, 1000, id)
			assert.Equal(t, expURL1, path)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &path))
			assert.Equal(t, 2000, id)
			assert.Equal(t, expURL2, path)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have converted the URLs back to local paths", func(t *testing.T) {
				rows, queryErr := eng.DB.Query(`SELECT id, local_path 
						FROM transfers ORDER BY id`)
				require.NoError(t, queryErr)

				defer rows.Close()

				var (
					id   int
					path string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &path))
				assert.Equal(t, 1000, id)
				assert.Equal(t, osPath1, path)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &path))
				assert.Equal(t, 2000, id)
				assert.Equal(t, osPath2, path)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_9_0FixLocalServerEnabled(t *testing.T, eng *testEngine) Change {
	mig := Migrations[39]

	t.Run("When applying the 0.9.0 local agent 'enable' column replacement", func(t *testing.T) {
		tableShouldHaveColumns(t, eng.DB, "local_agents", "enabled")
		tableShouldNotHaveColumns(t, eng.DB, "local_agents", "disabled")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have renamed the column", func(t *testing.T) {
			tableShouldNotHaveColumns(t, eng.DB, "local_agents", "enabled")
			tableShouldHaveColumns(t, eng.DB, "local_agents", "disabled")

			t.Run(`Then the "normalized_transfer" view should still	exist
				(sqlite only)`, func(t *testing.T) {
				if eng.Dialect == SQLite {
					err := eng.Exec(`SELECT * FROM normalized_transfers`)
					require.NoError(t, err)
				}
			})
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the column", func(t *testing.T) {
				tableShouldHaveColumns(t, eng.DB, "local_agents", "enabled")
				tableShouldNotHaveColumns(t, eng.DB, "local_agents", "disabled")

				t.Run(`Then the "normalized_transfer" view should still
						exist (sqlite only)`, func(t *testing.T) {
					if eng.Dialect == SQLite {
						err := eng.Exec(`SELECT * FROM normalized_transfers`)
						require.NoError(t, err)
					}
				})
			})
		})
	})

	return mig
}

func testVer0_9_0AddClientsTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[40]

	t.Run("When applying the 0.9.0 'clients' table creation", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO users(owner,username)
			VALUES ('bar','user_b1'), ('foo','user_f1'), ('bar','user_b2')`)

		eng.NoError(t, `INSERT INTO remote_agents(name,protocol,address)
    		VALUES ('p1','proto1','1.1.1.1'),
    		       ('p2','proto2','2.2.2.2'),
    		       ('p3','proto3','3.3.3.3')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM users`)
		})

		require.False(t, doesTableExist(t, eng.DB, eng.Dialect, "clients"))

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the table", func(t *testing.T) {
			require.True(t, doesTableExist(t, eng.DB, eng.Dialect, "clients"))
			tableShouldHaveColumns(t, eng.DB, "clients", "id", "owner", "name",
				"protocol", "disabled", "local_address", "proto_config")

			t.Run("Then it should have inserted 1 client per protocol", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,owner,name,protocol,
       				local_address,proto_config FROM clients ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id                               int
					owner, name, proto, addr, config string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 1, id)
				assert.Equal(t, "bar", owner)
				assert.Equal(t, "proto1", name)
				assert.Equal(t, "proto1", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 2, id)
				assert.Equal(t, "foo", owner)
				assert.Equal(t, "proto1", name)
				assert.Equal(t, "proto1", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 3, id)
				assert.Equal(t, "bar", owner)
				assert.Equal(t, "proto2", name)
				assert.Equal(t, "proto2", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 4, id)
				assert.Equal(t, "foo", owner)
				assert.Equal(t, "proto2", name)
				assert.Equal(t, "proto2", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 5, id)
				assert.Equal(t, "bar", owner)
				assert.Equal(t, "proto3", name)
				assert.Equal(t, "proto3", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &name, &proto, &addr, &config))
				assert.Equal(t, 6, id)
				assert.Equal(t, "foo", owner)
				assert.Equal(t, "proto3", name)
				assert.Equal(t, "proto3", proto)
				assert.Zero(t, addr)
				assert.Equal(t, "{}", config)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "clients"))
			})
		})
	})

	return mig
}

func testVer0_9_0AddRemoteAgentOwner(t *testing.T, eng *testEngine) Change {
	mig := Migrations[41]

	t.Run("When applying the 0.9.0 'owner' column addition", func(t *testing.T) {
		tableShouldNotHaveColumns(t, eng.DB, "remote_agents", "owner")

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "remote_agents", "owner")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "remote_agents", "owner")
			})
		})
	})

	return mig
}

//nolint:maintidx //function is complex because we must check lots of parameters
func testVer0_9_0DuplicateRemoteAgents(t *testing.T, eng *testEngine) Change {
	mig := Migrations[42]

	t.Run("When applying the 0.9.0 partner duplication", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO users(owner,username)
			VALUES ('aaa','user_a1'), ('bbb','user_b1'), ('bbb','user_b2')`)

		eng.NoError(t, `INSERT INTO
    		remote_agents(id,name,protocol,proto_config,address)
    		VALUES (1,'proto1_part','proto1','{}','addr1'),
    		       (2,'proto2_part','proto2','{}','addr2')`)

		eng.NoError(t, `INSERT INTO
    		remote_accounts(id,remote_agent_id,login,password)
    		VALUES (11,1,'proto1_acc','sesame1'),
    		       (12,2,'proto2_acc','sesame2')`)

		eng.NoError(t, `INSERT INTO rules(id,name,is_send,path)
			VALUES (10000,'push',true,'/push'), (20000,'pull',false,'/pull')`)

		eng.NoError(t, `INSERT INTO rule_access(remote_agent_id,remote_account_id,rule_id)
			VALUES (1,null,10000), (null,12,20000)`)

		eng.NoError(t, `INSERT INTO
			crypto_credentials(id,remote_agent_id,remote_account_id,name,
			                   private_key,ssh_public_key,certificate)
			VALUES (101,  1 ,null,'proto1_part_crypto','pk1','pbk1','cert1'),
			       (102,null, 12 ,'proto2_acc_crypto' ,'pk2','pbk2','cert2')`)

		// Pgsql does not increment the sequence when IDs are inserted manually,
		// so we have to manually increment the sequences to keep the test
		// consistent with other databases.
		if eng.Dialect == PostgreSQL {
			eng.NoError(t, `SELECT setval('remote_agents_id_seq', 2);
				SELECT setval('remote_accounts_id_seq', 12);
				SELECT setval('crypto_credentials_id_seq', 102)`)
		}

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM crypto_credentials`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM rule_access`)
			eng.NoError(t, `DELETE FROM rules`)
			eng.NoError(t, `DELETE FROM users`)
		})

		// The previous migration added an "owner" column to the remote_agents
		// table, so that partners may no longer be shared between GW instances.
		// To maintain the previous behavior, we need to duplicate all these
		// partners in the database, so that all instances which had access to
		// them can still use them. To do so, we query the "users" table to get
		// a list of all the gateway instances sharing this database, and then
		// for each of them, we create a copy of all known partners (and all of
		// their attached items).

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		var (
			duplProto1PartID, duplProto2PartID int
			duplProto2AccID                    int
		)

		t.Run("Then it should have duplicated the partners", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,owner,name,proto_config,
			       	address FROM remote_agents ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                        int
				owner, name, config, addr string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &config, &addr))
			assert.Equal(t, 1, id)
			assert.Equal(t, "aaa", owner)
			assert.Equal(t, "proto1_part", name)
			assert.Equal(t, "{}", config)
			assert.Equal(t, "addr1", addr)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &config, &addr))
			assert.Equal(t, 2, id)
			assert.Equal(t, "aaa", owner)
			assert.Equal(t, "proto2_part", name)
			assert.Equal(t, "{}", config)
			assert.Equal(t, "addr2", addr)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &config, &addr))
			assert.Equal(t, "bbb", owner)
			assert.Equal(t, "proto1_part", name)
			assert.Equal(t, "{}", config)
			assert.Equal(t, "addr1", addr)

			duplProto1PartID = id

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &name, &config, &addr))
			assert.Equal(t, "bbb", owner)
			assert.Equal(t, "proto2_part", name)
			assert.Equal(t, "{}", config)
			assert.Equal(t, "addr2", addr)

			duplProto2PartID = id

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("Then it should have duplicated the accounts", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,remote_agent_id,login,password
       				FROM remote_accounts ORDER BY remote_agent_id,login`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id, partnerID   int
				login, password string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &partnerID, &login, &password))
			assert.Equal(t, 11, id)
			assert.Equal(t, 1, partnerID)
			assert.Equal(t, "proto1_acc", login)
			assert.Equal(t, "sesame1", password)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &partnerID, &login, &password))
			assert.Equal(t, 12, id)
			assert.Equal(t, 2, partnerID)
			assert.Equal(t, "proto2_acc", login)
			assert.Equal(t, "sesame2", password)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &partnerID, &login, &password))
			assert.Equal(t, duplProto1PartID, partnerID)
			assert.Equal(t, "proto1_acc", login)
			assert.Equal(t, "sesame1", password)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &partnerID, &login, &password))
			assert.Equal(t, duplProto2PartID, partnerID)
			assert.Equal(t, "proto2_acc", login)
			assert.Equal(t, "sesame2", password)

			duplProto2AccID = id

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("Then it should have duplicated the crypto credentials", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,remote_agent_id,remote_account_id,
       				name,private_key,ssh_public_key,certificate FROM crypto_credentials
       				ORDER BY remote_account_id IS NOT NULL, remote_account_id,
       				         remote_agent_id IS NOT NULL, remote_agent_id, name`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                  int
				remAgID, remAccID   sql.NullInt64
				name, pk, pbk, cert string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remAgID, &remAccID, &name, &pk, &pbk, &cert))
			assert.Equal(t, 101, id)
			assert.Equal(t, int64(1), remAgID.Int64)
			assert.Zero(t, remAccID.Int64)
			assert.Equal(t, "proto1_part_crypto", name)
			assert.Equal(t, "pk1", pk)
			assert.Equal(t, "pbk1", pbk)
			assert.Equal(t, "cert1", cert)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remAgID, &remAccID, &name, &pk, &pbk, &cert))
			assert.Equal(t, int64(duplProto1PartID), remAgID.Int64)
			assert.Zero(t, remAccID.Int64)
			assert.Equal(t, "proto1_part_crypto", name)
			assert.Equal(t, "pk1", pk)
			assert.Equal(t, "pbk1", pbk)
			assert.Equal(t, "cert1", cert)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remAgID, &remAccID, &name, &pk, &pbk, &cert))
			assert.Equal(t, 102, id)
			assert.Zero(t, remAgID.Int64)
			assert.Equal(t, int64(12), remAccID.Int64)
			assert.Equal(t, "proto2_acc_crypto", name)
			assert.Equal(t, "pk2", pk)
			assert.Equal(t, "pbk2", pbk)
			assert.Equal(t, "cert2", cert)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remAgID, &remAccID, &name, &pk, &pbk, &cert))
			assert.Zero(t, remAgID.Int64)
			assert.Equal(t, int64(duplProto2AccID), remAccID.Int64)
			assert.Equal(t, "proto2_acc_crypto", name)
			assert.Equal(t, "pk2", pk)
			assert.Equal(t, "pbk2", pbk)
			assert.Equal(t, "cert2", cert)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("Then it should have duplicated the rule accesses", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT remote_agent_id,remote_account_id,
				rule_id FROM rule_access ORDER BY rule_id,
			  		remote_account_id IS NOT NULL, remote_account_id,
					remote_agent_id IS NOT NULL, remote_agent_id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				remAgID, remAccID sql.NullInt64
				ruleID            int
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&remAgID, &remAccID, &ruleID))
			assert.Equal(t, int64(1), remAgID.Int64)
			assert.Zero(t, remAccID.Int64)
			assert.Equal(t, 10000, ruleID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&remAgID, &remAccID, &ruleID))
			assert.Equal(t, int64(duplProto1PartID), remAgID.Int64)
			assert.Zero(t, remAccID.Int64)
			assert.Equal(t, 10000, ruleID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&remAgID, &remAccID, &ruleID))
			assert.Zero(t, remAgID.Int64)
			assert.Equal(t, int64(12), remAccID.Int64)
			assert.Equal(t, 20000, ruleID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&remAgID, &remAccID, &ruleID))
			assert.Zero(t, remAgID.Int64)
			assert.Equal(t, int64(duplProto2AccID), remAccID.Int64)
			assert.Equal(t, 20000, ruleID)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have removed the duplicated partners", func(t *testing.T) {
				var id int

				rows, err := eng.DB.Query(`SELECT id FROM remote_agents ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 1, id)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 2, id)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})

			t.Run("Then it should have removed the duplicated accounts", func(t *testing.T) {
				var id int

				rows, err := eng.DB.Query(`SELECT id FROM remote_accounts ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 11, id)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 12, id)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})

			t.Run("Then it should have removed the duplicated cryptos", func(t *testing.T) {
				var id int

				rows, err := eng.DB.Query(`SELECT id FROM crypto_credentials ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 101, id)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id))
				assert.Equal(t, 102, id)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})

			t.Run("Then it should have removed the duplicated rule accesses", func(t *testing.T) {
				var remAgID, remAccID sql.NullInt64

				rows, err := eng.DB.Query(`SELECT remote_agent_id,remote_account_id
						FROM rule_access ORDER BY rule_id,remote_account_id,remote_agent_id`)
				require.NoError(t, err)

				defer rows.Close()

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remAgID, &remAccID))
				assert.Equal(t, int64(1), remAgID.Int64)
				assert.Zero(t, remAccID.Int64)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&remAgID, &remAccID))
				assert.Zero(t, remAgID.Int64)
				assert.Equal(t, int64(12), remAccID.Int64)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_9_0RelinkTransfers(t *testing.T, eng *testEngine) Change {
	mig := Migrations[43]

	t.Run("When applying the 0.9.0 transfer agent relink", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO users(owner,username)
			VALUES ('aaa','user_a1'), ('bbb','user_b1'), ('bbb','user_b2')`)

		eng.NoError(t, `INSERT INTO
    		remote_agents(id,owner,name,protocol,address)
    		VALUES (10, 'aaa','proto1_partner', 'proto1', '1.1.1.1'),
    		       (20, 'aaa','proto2_partner', 'proto2', '2.2.2.2'),
    		       (30, 'bbb','proto1_partner', 'proto1', '1.1.1.1'),
    		       (40, 'bbb','proto2_partner', 'proto2', '2.2.2.2')`)

		eng.NoError(t, `INSERT INTO
    		remote_accounts(id,remote_agent_id,login)
    		VALUES (100, 10, 'proto1_account'), (200, 20, 'proto2_account'),
    		       (300, 30, 'proto1_account'), (400, 40, 'proto2_account')`)

		eng.NoError(t, `INSERT INTO rules(id,name,is_send,path)
			VALUES (1000,'push',true,'/push')`)

		eng.NoError(t, `INSERT INTO transfers(id,owner,remote_transfer_id,
                rule_id,remote_account_id,local_path,remote_path)
            VALUES (10000, 'aaa', 'proto1a', 1000, 100, '/loc/path', '/rem/path'),
                   (20000, 'aaa', 'proto2a', 1000, 200, '/loc/path', '/rem/path'),
                   (30000, 'bbb', 'proto1b', 1000, 100, '/loc/path', '/rem/path'),
                   (40000, 'bbb', 'proto2b', 1000, 200, '/loc/path', '/rem/path')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfers`)
			eng.NoError(t, `DELETE FROM rules`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM users`)
		})

		// The previous migration duplicated all remote agents in the database
		// (as well as all of it's associated objects like remote accounts).
		// However, all transfers are still linked to remote accounts
		// (and subsequently to remote agents) owned by the first GW instance
		// in alphabetical order (in this case "aaa"), even though some
		// transfers were not performed by that GW instance (as evidenced by
		// the "owner" column). This migration relinks these transfers to their
		// correct remote account.

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have updated the relevant transfers", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,owner,remote_account_id
					FROM transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id, remAccID int
				owner        string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remAccID))
			assert.Equal(t, 10000, id)
			assert.Equal(t, "aaa", owner)
			assert.Equal(t, 100, remAccID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remAccID))
			assert.Equal(t, 20000, id)
			assert.Equal(t, "aaa", owner)
			assert.Equal(t, 200, remAccID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remAccID))
			assert.Equal(t, 30000, id)
			assert.Equal(t, "bbb", owner)
			assert.Equal(t, 300, remAccID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &owner, &remAccID))
			assert.Equal(t, 40000, id)
			assert.Equal(t, "bbb", owner)
			assert.Equal(t, 400, remAccID)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			// When reverting the migration, all transfers are re-attributed
			// to their original remote account owned by the first GW instance
			// in alphabetical order (here "aaa").

			t.Run("Then it should have reverted the changes", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,owner,remote_account_id
					FROM transfers ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id, remAccID int
					owner        string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &remAccID))
				assert.Equal(t, 10000, id)
				assert.Equal(t, "aaa", owner)
				assert.Equal(t, 100, remAccID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &remAccID))
				assert.Equal(t, 20000, id)
				assert.Equal(t, "aaa", owner)
				assert.Equal(t, 200, remAccID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &remAccID))
				assert.Equal(t, 30000, id)
				assert.Equal(t, "bbb", owner)
				assert.Equal(t, 100, remAccID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &owner, &remAccID))
				assert.Equal(t, 40000, id)
				assert.Equal(t, "bbb", owner)
				assert.Equal(t, 200, remAccID)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

func testVer0_9_0AddTransfersClientID(t *testing.T, eng *testEngine) Change {
	mig := Migrations[44]

	t.Run("When applying the 0.9.0 transfer client_id addition", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO clients (id, name, owner, protocol)
			VALUES (1, 'proto1', 'aaa', 'proto1'), (2, 'proto2', 'aaa', 'proto2'),
			       (3, 'proto1', 'bbb', 'proto1'), (4, 'proto2', 'bbb', 'proto2')`)

		eng.NoError(t,
			`INSERT INTO remote_agents(id, name, owner, protocol, address)
			VALUES (10, 'partner1', 'aaa', 'proto1', 'addr1'),
			       (20, 'partner2', 'aaa', 'proto2', 'addr2'),
			       (30, 'partner1', 'bbb', 'proto1', 'addr1'),
			       (40, 'partner2', 'bbb', 'proto2', 'addr2')`)

		eng.NoError(t, `INSERT INTO
    		remote_accounts(id, remote_agent_id, login)
    		VALUES (100, 10, 'account1'), (200, 20, 'account2'),
    		       (300, 30, 'account1'), (400, 40, 'account2')`)

		eng.NoError(t, `INSERT INTO rules(id, name, is_send, path)
			VALUES (1000, 'push', true, '/push')`)

		eng.NoError(t, `INSERT INTO transfers(id, owner, remote_transfer_id,
                rule_id, remote_account_id, local_path, remote_path)
            VALUES (10000, 'aaa', 'proto1a', 1000, 100, '/loc/path', '/rem/path'),
                   (20000, 'aaa', 'proto2a', 1000, 200, '/loc/path', '/rem/path'),
                   (30000, 'bbb', 'proto1b', 1000, 300, '/loc/path', '/rem/path'),
                   (40000, 'bbb', 'proto2b', 1000, 400, '/loc/path', '/rem/path')`)

		tableShouldNotHaveColumns(t, eng.DB, "transfers", "client_id")

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfers`)
			eng.NoError(t, `DELETE FROM rules`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM clients`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added and filled the 'client_id' column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfers", "client_id")

			rows, err := eng.DB.Query(`SELECT id, client_id
					FROM transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var id, clientID int

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &clientID))
			assert.Equal(t, 10000, id)
			assert.Equal(t, 1, clientID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &clientID))
			assert.Equal(t, 20000, id)
			assert.Equal(t, 2, clientID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &clientID))
			assert.Equal(t, 30000, id)
			assert.Equal(t, 3, clientID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &clientID))
			assert.Equal(t, 40000, id)
			assert.Equal(t, 4, clientID)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the 'client_id' column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfers", "client_id")
			})
		})
	})

	return mig
}

func testVer0_9_0AddHistoryClient(t *testing.T, eng *testEngine) Change {
	mig := Migrations[45]

	t.Run("When applying the 0.9.0 history client addition", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO transfer_history(id,owner,
        	remote_transfer_id,is_server,is_send,rule,account,agent,
            protocol,local_path,remote_path,start,stop,status,step) VALUES
            (1,'wg','abc',true,true,'push','loc_ag','loc_acc','proto1','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone'),
        	(2,'wg','def',false,true,'push','rem_ag1','rem_acc1','proto1','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone'),
            (3,'wg','ghi',false,false,'pull','rem_ag2','rem_acc2','proto2','/loc/path',
             '/rem/path','2022-01-01 01:00:00','2022-01-01 02:00:00','DONE','StepNone')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfer_history`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added and filled the 'client' column", func(t *testing.T) {
			tableShouldHaveColumns(t, eng.DB, "transfer_history", "client")

			rows, err := eng.DB.Query(`SELECT id, client FROM transfer_history
					ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id     int
				client string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &client))
			assert.Equal(t, 1, id)
			assert.Zero(t, client)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &client))
			assert.Equal(t, 2, id)
			assert.Equal(t, "proto1_client", client)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &client))
			assert.Equal(t, 3, id)
			assert.Equal(t, "proto2_client", client)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the 'client' column", func(t *testing.T) {
				tableShouldNotHaveColumns(t, eng.DB, "transfer_history", "client")
			})
		})
	})

	return mig
}

func testVer0_9_0AddNormalizedTransfersView(t *testing.T, eng *testEngine) Change {
	mig := Migrations[46]

	t.Run("When applying the 0.9.0 normalized transfer view restoration", func(t *testing.T) {
		// ### CLIENTS ###
		eng.NoError(t, `INSERT INTO clients(id, owner, name, protocol)
			VALUES (2222, 'bbb', 'sftp', 'sftp')`)

		// ### RULES ###
		eng.NoError(t, `INSERT INTO rules(id, name, is_send, path)
			VALUES (1, 'push', TRUE, '/push'), (2, 'pull', FALSE, '/pull')`)

		// ### LOCAL ###
		eng.NoError(t, `INSERT INTO local_agents(id,owner,name,protocol,address)
			VALUES (10, 'aaa', 'sftp_serv', 'sftp', '1.1.1.1:1111')`)

		eng.NoError(t, `INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100, 10, 'toto')`)

		// ### REMOTE ###
		eng.NoError(t, `INSERT INTO remote_agents(id,owner,name,protocol,address)
			VALUES (20, 'bbb', 'sftp_part', 'sftp', '2.2.2.2:2222')`)

		eng.NoError(t, `INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200, 20, 'tata')`)

		// ### TRANSFERS ###
		eng.NoError(t, `INSERT INTO transfers(id,owner,remote_transfer_id,rule_id,
            client_id,local_account_id,remote_account_id,src_filename,dest_filename)
            VALUES(1000, 'aaa', 'abcd', 1, NULL, 100, NULL, '/src/1', '/dst/1'),
                  (2000, 'bbb', 'efgh', 2, 2222, NULL, 200, '/src/2', '/dst/2')`)

		eng.NoError(t, `INSERT INTO transfer_history(id,owner,is_server,
            	is_send,remote_transfer_id,rule,client,account,agent,protocol,
                src_filename,dest_filename,filesize,start,stop,status,step)
			VALUES (3000,'ccc',FALSE,TRUE,'xyz','push','r66_client','tutu',
				'r66_part','r66','/src/3','/dst/3',123,'2021-01-03 01:00:00',
			    '2021-01-03 02:00:00','CANCELLED','StepData')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfer_history`)
			eng.NoError(t, `DELETE FROM transfers`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM local_accounts`)
			eng.NoError(t, `DELETE FROM local_agents`)
			eng.NoError(t, `DELETE FROM rules`)
			eng.NoError(t, `DELETE FROM clients`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the view", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,is_server,is_send,is_transfer,
       				rule,client,account,agent,protocol,src_filename,dest_filename
					FROM normalized_transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                                  int
				isServ, isSend, isTrans             bool
				rule, client, account, agent, proto string
				srcFile, dstFile                    string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &isServ, &isSend, &isTrans, &rule,
				&client, &account, &agent, &proto, &srcFile, &dstFile))
			assert.Equal(t, 1000, id)
			assert.True(t, isServ)
			assert.True(t, isSend)
			assert.True(t, isTrans)
			assert.Equal(t, "push", rule)
			assert.Equal(t, "", client)
			assert.Equal(t, "toto", account)
			assert.Equal(t, "sftp_serv", agent)
			assert.Equal(t, "sftp", proto)
			assert.Equal(t, "/src/1", srcFile)
			assert.Equal(t, "/dst/1", dstFile)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &isServ, &isSend, &isTrans, &rule,
				&client, &account, &agent, &proto, &srcFile, &dstFile))
			assert.Equal(t, 2000, id)
			assert.False(t, isServ)
			assert.False(t, isSend)
			assert.True(t, isTrans)
			assert.Equal(t, "pull", rule)
			assert.Equal(t, "sftp", client)
			assert.Equal(t, "tata", account)
			assert.Equal(t, "sftp_part", agent)
			assert.Equal(t, "sftp", proto)
			assert.Equal(t, "/src/2", srcFile)
			assert.Equal(t, "/dst/2", dstFile)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &isServ, &isSend, &isTrans, &rule,
				&client, &account, &agent, &proto, &srcFile, &dstFile))
			assert.Equal(t, 3000, id)
			assert.False(t, isServ)
			assert.True(t, isSend)
			assert.False(t, isTrans)
			assert.Equal(t, "push", rule)
			assert.Equal(t, "r66_client", client)
			assert.Equal(t, "tutu", account)
			assert.Equal(t, "r66_part", agent)
			assert.Equal(t, "r66", proto)
			assert.Equal(t, "/src/3", srcFile)
			assert.Equal(t, "/dst/3", dstFile)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the view", func(t *testing.T) {
				err := eng.Exec(`SELECT * FROM normalized_transfers`)
				shouldBeTableNotExist(t, err)
			})
		})
	})

	return mig
}

func testVer0_9_0AddCredTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[47]

	t.Run("When applying the 0.9.0 'credentials' table creation", func(t *testing.T) {
		assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "credentials"))

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the table", func(t *testing.T) {
			assert.True(t, doesTableExist(t, eng.DB, eng.Dialect, "credentials"))
			tableShouldHaveColumns(t, eng.DB, "credentials",
				"local_agent_id", "remote_agent_id", "local_account_id", "remote_account_id",
				"name", "type", "value", "value2")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the table", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "credentials"))
			})
		})
	})

	return mig
}

func testVer0_9_0FillCredTable(t *testing.T, eng *testEngine) Change {
	mig := Migrations[48]

	t.Run("When applying the 0.9.0 'credentials' table filling", func(t *testing.T) {
		_ver0_9_0FillAuthTableIgnoreCertParseError = true
		compatibility.IsLegacyR66CertificateAllowed = true

		defer func() {
			_ver0_9_0FillAuthTableIgnoreCertParseError = false
			compatibility.IsLegacyR66CertificateAllowed = false
		}()

		// ### Local agent ###
		eng.NoError(t, `INSERT INTO local_agents(id,owner,name,protocol,address)
			VALUES (10,'waarp_gw','sftp_serv','sftp','localhost:2222')`)

		// ### Local account ###
		eng.NoError(t, `INSERT INTO local_accounts(id,local_agent_id,login,
        	password_hash) VALUES (100,10,'toto','pswd_hash')`)

		// ### Remote agent ###
		eng.NoError(t, `INSERT INTO remote_agents(id,name,protocol,address)
			VALUES (20,'sftp_part','sftp','localhost:3333')`)

		// ### Remote account ###
		eng.NoError(t, `INSERT INTO remote_accounts(id,remote_agent_id,login,
            password) VALUES (200,20,'toto','$AES$pswd')`)

		// ##### Certificates #####
		//nolint:dupword //NULL is repeated on purpose
		eng.NoError(t, `INSERT INTO crypto_credentials
    		(id,name,local_agent_id,remote_agent_id,local_account_id,remote_account_id,
    		 private_key,certificate,ssh_public_key) 
			VALUES (1010, '1-lag_cert',     10,   NULL, NULL, NULL, '$AES$pk1',    'cert1', ''),
			       (1020, '2-rag_cert',     NULL, 20,   NULL, NULL, '',            'cert2', ''),
			       (1100, '3-lac_ssh',      NULL, NULL, 100,  NULL, '',            '',      'ssh_pbk1'),
			       (1200, '4-rac_ssh',      NULL, NULL, NULL, 200, '$AES$ssh_pk1', '',      ''),
			       (1011, '5-lag_r66_cert', 10,   NULL, NULL, NULL, ?,             ?,       ''),
			       (1021, '6-rag_r66_cert', NULL, 20,   NULL, NULL, '',            ?,       '')`,
			compatibility.LegacyR66KeyPEM, compatibility.LegacyR66CertPEM,
			compatibility.LegacyR66CertPEM)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM crypto_credentials`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM local_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM local_agents`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have filled the table", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT local_agent_id,remote_agent_id,
       			local_account_id,remote_account_id,name,type,value,value2 
				FROM credentials ORDER BY name,type,id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				lagID, ragID, lacID, racID sql.NullInt64
				name, typ, val, val2       string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Equal(t, utils.NewNullInt64(10), lagID)
			assert.Zero(t, ragID)
			assert.Zero(t, lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "1-lag_cert", name)
			assert.Equal(t, "tls_certificate", typ)
			assert.Equal(t, "cert1", val)
			assert.Equal(t, "pk1", val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Equal(t, utils.NewNullInt64(20), ragID)
			assert.Zero(t, lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "2-rag_cert", name)
			assert.Equal(t, "trusted_tls_certificate", typ)
			assert.Equal(t, "cert2", val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Zero(t, ragID)
			assert.Equal(t, utils.NewNullInt64(100), lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "3-lac_ssh", name)
			assert.Equal(t, "ssh_public_key", typ)
			assert.Equal(t, "ssh_pbk1", val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Zero(t, ragID)
			assert.Zero(t, lacID)
			assert.Equal(t, utils.NewNullInt64(200), racID)
			assert.Equal(t, "4-rac_ssh", name)
			assert.Equal(t, "ssh_private_key", typ)
			assert.Equal(t, "ssh_pk1", val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Equal(t, utils.NewNullInt64(10), lagID)
			assert.Zero(t, ragID)
			assert.Zero(t, lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "5-lag_r66_cert", name)
			assert.Equal(t, "r66_legacy_certificate", typ)
			assert.Zero(t, val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Equal(t, utils.NewNullInt64(20), ragID)
			assert.Zero(t, lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "6-rag_r66_cert", name)
			assert.Equal(t, "r66_legacy_certificate", typ)
			assert.Zero(t, val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Zero(t, ragID)
			assert.Equal(t, utils.NewNullInt64(100), lacID)
			assert.Zero(t, racID)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pswd_hash", val)
			assert.Zero(t, val2)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &typ, &val, &val2))
			assert.Zero(t, lagID)
			assert.Zero(t, ragID)
			assert.Zero(t, lacID)
			assert.Equal(t, utils.NewNullInt64(200), racID)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pswd", val)
			assert.Zero(t, val2)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the table", func(t *testing.T) {
				row := eng.DB.QueryRow("SELECT COUNT(*) FROM credentials")

				var count int64
				require.NoError(t, row.Scan(&count))
				assert.Zero(t, count)
			})
		})
	})

	return mig
}

func testVer0_9_0RemoveOldCreds(t *testing.T, eng *testEngine) Change {
	mig := Migrations[49]

	t.Run("When applying the 0.9.0 'crypto_credentials' table removal", func(t *testing.T) {
		// ### Local agent ###
		eng.NoError(t, `INSERT INTO local_agents(id,owner,name,protocol,
            address) VALUES (10,'waarp_gw','http_serv','http','localhost:2222')`)

		// ### Local account ###
		eng.NoError(t, `INSERT INTO local_accounts(id,local_agent_id,login,
        	password_hash) VALUES (100,10,'toto','pswd_hash')`)

		// ### Remote agent ###
		eng.NoError(t, `INSERT INTO remote_agents(id,name,protocol,address)
			VALUES (20,'sftp_part','sftp','localhost:3333')`)

		// ### Remote account ###
		eng.NoError(t, `INSERT INTO remote_accounts(id,remote_agent_id,login,
            password) VALUES (200,20,'toto','$AES$pswd')`)

		// ##### Credentials #####
		eng.NoError(t, `INSERT INTO credentials
    		(local_agent_id,remote_agent_id,local_account_id,remote_account_id,
    		 name,type,value,value2) VALUES
			( 10 ,NULL,NULL,NULL,'1_lag_cert','tls_certificate','cert1','pk1'),
			(NULL, 20 ,NULL,NULL,'2_rag_pblk','ssh_public_key','pbk2',''),
			(NULL,NULL,100 ,NULL,'3_lac_cert','trusted_tls_certificate','cert3',''),
			(NULL,NULL,NULL,200 ,'4_rac_prvk','ssh_private_key','pk4',''),
			(NULL,NULL,100 ,NULL,'5_lac_pswd','password','pswd_hash',''),
			(NULL,NULL,NULL,200 ,'6_rac_pswd','password','pswd',''),
			( 10 ,NULL,NULL,NULL,'7_lag_r66_cert','r66_legacy_certificate','',''),
			(NULL, 20 ,NULL,NULL,'8_rag_r66_cert','r66_legacy_certificate','','')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM credentials")
			eng.NoError(t, "DELETE FROM remote_accounts")
			eng.NoError(t, "DELETE FROM local_accounts")
			eng.NoError(t, "DELETE FROM remote_agents")
			eng.NoError(t, "DELETE FROM local_agents")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have deleted the 'crypto_credentials' table", func(t *testing.T) {
			assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "crypto_credentials"))
		})

		t.Run("Then it should have deleted the account password columns", func(t *testing.T) {
			tableShouldNotHaveColumns(t, eng.DB, "local_accounts", "password_hash")
			tableShouldNotHaveColumns(t, eng.DB, "remote_accounts", "password")
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the 'crypto_credentials' table", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT local_agent_id,remote_agent_id,
   					local_account_id,remote_account_id,name,certificate,private_key,
   					ssh_public_key FROM crypto_credentials ORDER BY name`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					lagID, ragID, lacID, racID sql.NullInt64
					name, cert, pk, pbk        string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Equal(t, utils.NewNullInt64(10), lagID)
				assert.Zero(t, ragID)
				assert.Zero(t, lacID)
				assert.Zero(t, racID)
				assert.Equal(t, "1_lag_cert", name)
				assert.Equal(t, "cert1", cert)
				assert.Equal(t, "$AES$pk1", pk)
				assert.Zero(t, pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Zero(t, lagID)
				assert.Equal(t, utils.NewNullInt64(20), ragID)
				assert.Zero(t, lacID)
				assert.Zero(t, racID)
				assert.Equal(t, "2_rag_pblk", name)
				assert.Zero(t, cert)
				assert.Zero(t, pk)
				assert.Equal(t, "pbk2", pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Zero(t, lagID)
				assert.Zero(t, ragID)
				assert.Equal(t, utils.NewNullInt64(100), lacID)
				assert.Zero(t, racID)
				assert.Equal(t, "3_lac_cert", name)
				assert.Equal(t, "cert3", cert)
				assert.Zero(t, pk)
				assert.Zero(t, pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Zero(t, lagID)
				assert.Zero(t, ragID)
				assert.Zero(t, lacID)
				assert.Equal(t, utils.NewNullInt64(200), racID)
				assert.Equal(t, "4_rac_prvk", name)
				assert.Zero(t, cert)
				assert.Equal(t, "$AES$pk4", pk)
				assert.Zero(t, pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Equal(t, utils.NewNullInt64(10), lagID)
				assert.Zero(t, ragID)
				assert.Zero(t, lacID)
				assert.Zero(t, racID)
				assert.Equal(t, "7_lag_r66_cert", name)
				assert.Equal(t, compatibility.LegacyR66CertPEM, cert)
				assert.Equal(t, "$AES$"+compatibility.LegacyR66KeyPEM, pk)
				assert.Zero(t, pbk)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&lagID, &ragID, &lacID, &racID, &name, &cert, &pk, &pbk))
				assert.Zero(t, lagID)
				assert.Equal(t, utils.NewNullInt64(20), ragID)
				assert.Zero(t, lacID)
				assert.Zero(t, racID)
				assert.Equal(t, "8_rag_r66_cert", name)
				assert.Equal(t, compatibility.LegacyR66CertPEM, cert)
				assert.Zero(t, pk)
				assert.Zero(t, pbk)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})

			t.Run("Then it should have restored the local account password column", func(t *testing.T) {
				tableShouldHaveColumns(t, eng.DB, "local_accounts", "password_hash")

				query := `SELECT password_hash FROM local_accounts WHERE id=100`
				row := eng.DB.QueryRow(query)

				var hash string
				require.NoError(t, row.Scan(&hash))
				assert.Equal(t, "pswd_hash", hash)
			})

			t.Run("Then it should have restored the remote account password column", func(t *testing.T) {
				tableShouldHaveColumns(t, eng.DB, "remote_accounts", "password")

				query := `SELECT password FROM remote_accounts WHERE id=200`
				row := eng.DB.QueryRow(query)

				var pswd string
				require.NoError(t, row.Scan(&pswd))
				assert.Equal(t, "$AES$pswd", pswd)
			})
		})
	})

	return mig
}

func testVer0_9_0MoveR66ServerCreds(t *testing.T, eng *testEngine) Change {
	mig := Migrations[50]

	t.Run("When applying the 0.9.0 R66 credentials extraction", func(t *testing.T) {
		// ### Local agents ###
		eng.NoError(t, `
			INSERT INTO local_agents(id,owner,name,protocol,address,proto_config)
			VALUES (1,'waarp_gw','gw_r66_1','r66','localhost:1',
			        '{"serverLogin":"gw_1","serverPassword":"$AES$pwd1"}'),
			       (2,'waarp_gw','gw_r66_2','r66','localhost:2',
			        '{"serverLogin":"gw_2","serverPassword":"$AES$pwd2"}')`)

		// ### Remote agents ###
		eng.NoError(t, `
			INSERT INTO remote_agents(id,name,protocol,address,proto_config)
			VALUES (3,'waarp_r66_1','r66','localhost:3',
			        '{"serverLogin":"wr66_1","serverPassword":"pwd_hash3"}'),
			       (4,'waarp_r66_2','r66','localhost:4',
			        '{"serverLogin":"wr66_2","serverPassword":"pwd_hash4"}')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM local_agents`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have extracted the credentials from the proto config", func(t *testing.T) {
			rows, err := eng.DB.Query(`
					SELECT local_agents.id AS id,proto_config,credentials.name,type,value
					FROM local_agents LEFT JOIN credentials ON local_agent_id = local_agents.id
					UNION ALL
					SELECT remote_agents.id AS id,proto_config,credentials.name,type,value
					FROM remote_agents LEFT JOIN credentials ON remote_agent_id = remote_agents.id
					ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id                   int
				conf, name, typ, val string
				checkConf            = func(tb testing.TB) {
					tb.Helper()

					var parsedConf map[string]any

					require.NoError(t, json.Unmarshal([]byte(conf), &parsedConf))
					assert.NotContains(t, parsedConf, "serverPassword")
				}
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &conf, &name, &typ, &val))
			assert.Equal(t, 1, id)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pwd1", val)
			checkConf(t)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &conf, &name, &typ, &val))
			assert.Equal(t, 2, id)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pwd2", val)
			checkConf(t)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &conf, &name, &typ, &val))
			assert.Equal(t, 3, id)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pwd_hash3", val)
			checkConf(t)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &conf, &name, &typ, &val))
			assert.Equal(t, 4, id)
			assert.Equal(t, "password", name)
			assert.Equal(t, "password", typ)
			assert.Equal(t, "pwd_hash4", val)
			checkConf(t)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have reinserted the credentials in the proto config", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,proto_config FROM local_agents
						UNION ALL SELECT id,proto_config FROM remote_agents
						ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id        int
					conf      string
					checkConf = func(tb testing.TB, expectedServPswd string) {
						tb.Helper()

						var parsedConf map[string]any

						require.NoError(t, json.Unmarshal([]byte(conf), &parsedConf))
						assert.Contains(t, parsedConf, "serverPassword")
						assert.Equal(t, expectedServPswd, parsedConf["serverPassword"])
					}
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &conf))
				assert.Equal(t, 1, id)
				checkConf(t, "$AES$pwd1")

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &conf))
				assert.Equal(t, 2, id)
				checkConf(t, "$AES$pwd2")

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &conf))
				assert.Equal(t, 3, id)
				checkConf(t, "pwd_hash3")

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &conf))
				assert.Equal(t, 4, id)
				checkConf(t, "pwd_hash4")

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})

			t.Run("Then it should have deleted the credentials entries", func(t *testing.T) {
				row := eng.DB.QueryRow(`SELECT * FROM credentials`)
				require.ErrorIs(t, row.Scan(), sql.ErrNoRows)
			})
		})
	})

	return mig
}

func testVer0_9_0AddAuthorities(t *testing.T, eng *testEngine) Change {
	mig := Migrations[51]

	t.Run("When applying the 0.9.0 'authorities' table creation", func(t *testing.T) {
		assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "auth_authorities"))
		assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "authority_hosts"))

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have added the tables", func(t *testing.T) {
			require.True(t, doesTableExist(t, eng.DB, eng.Dialect, "auth_authorities"))
			require.True(t, doesTableExist(t, eng.DB, eng.Dialect, "authority_hosts"))

			tableShouldHaveColumns(t, eng.DB, "auth_authorities",
				"id", "name", "type", "public_identity")
			tableShouldHaveColumns(t, eng.DB, "authority_hosts",
				"authority_id", "host")
		})

		t.Run("When reversing the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have dropped the tables", func(t *testing.T) {
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "auth_authorities"))
				assert.False(t, doesTableExist(t, eng.DB, eng.Dialect, "authority_hosts"))
			})
		})
	})

	return mig
}
