package backup

import (
	"testing"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
	r66lib "code.waarp.fr/lib/r66"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportLocalAgents(t *testing.T) {
	t.Parallel()

	setup := func(tb testing.TB) (*database.DB, *log.Logger, *model.LocalAgent, *model.LocalAgent) {
		tb.Helper()

		db := dbtest.TestDatabase(t)
		logger := nolog(t)

		agent := &model.LocalAgent{
			Name: "server", Protocol: testProtocol,
			Address: types.Addr("localhost", 2022),
		}
		other := &model.LocalAgent{
			Name: "other", Protocol: testProtocol,
			Address: types.Addr("localhost", 8888),
		}
		require.NoError(t, db.Insert(agent).Run())
		require.NoError(t, db.Insert(other).Run())

		service1, err := protocols.MakeServer(db, agent)
		require.NoError(tb, err)
		require.NoError(t, service1.Start())
		service2, err := protocols.MakeServer(db, other)
		require.NoError(tb, err)
		require.NoError(t, service2.Start())

		services.Servers.Add(agent, service1)
		services.Servers.Add(other, service2)

		return db, logger, agent, other
	}

	t.Run("Insert new", func(t *testing.T) {
		db, logger, agent, other := setup(t)

		newServer := LocalAgent{
			Name:          "foo",
			Protocol:      testProtocol,
			Configuration: map[string]any{},
			Address:       "localhost:1111",
			Accounts: []LocalAccount{
				{
					Login:    "acc1",
					Password: "pwd",
				}, {
					Login:    "acc2",
					Password: "pwd",
				},
			},
		}
		newServers := []LocalAgent{newServer}

		t.Run("No reset", func(t *testing.T) {
			require.NoError(t, importLocalAgents(logger, db, newServers, false, true))

			var dbAgents model.LocalAgents
			require.NoError(t, db.Select(&dbAgents).OrderBy("id", true).Run())
			require.Len(t, dbAgents, 3)
			dbAgent := dbAgents[2]

			t.Run("Check servers", func(t *testing.T) {
				assert.Equal(t, agent, dbAgents[0])
				assert.Equal(t, other, dbAgents[1])

				assert.Equal(t, newServer.Name, dbAgent.Name)
				assert.Equal(t, newServer.Protocol, dbAgent.Protocol)
				assert.Equal(t, newServer.Configuration, dbAgent.ProtoConfig.Map())
			})

			t.Run("Check accounts", func(t *testing.T) {
				var accounts model.LocalAccounts
				require.NoError(t, db.Select(&accounts).Where("local_agent_id=?", dbAgent.ID).Run())
				require.Len(t, accounts, 2)

				assert.Equal(t, newServer.Accounts[0].Login, accounts[0].Login)
				assert.Equal(t, newServer.Accounts[1].Login, accounts[1].Login)
			})

			t.Run("Check services", func(t *testing.T) {
				state, exists := services.Servers.State(dbAgent)
				require.True(t, exists)
				assert.Equal(t, utils.StateRunning, state)
			})
		})

		t.Run("With reset", func(t *testing.T) {
			require.NoError(t, importLocalAgents(logger, db, newServers, true, false))

			var dbAgents model.LocalAgents
			require.NoError(t, db.Select(&dbAgents).OrderBy("id", true).Run())
			require.Len(t, dbAgents, 1)

			dbAgent := dbAgents[0]

			assert.Equal(t, newServer.Name, dbAgent.Name)
			assert.Equal(t, newServer.Protocol, dbAgent.Protocol)
			assert.Equal(t, newServer.Configuration, dbAgent.ProtoConfig.Map())
		})
	})

	t.Run("Update existing", func(t *testing.T) {
		db, logger, agent, other := setup(t)

		updatedAgent := LocalAgent{
			Name:          agent.Name,
			Protocol:      testProtocol,
			Configuration: map[string]any{},
			Address:       "localhost:6666",
			Accounts: []LocalAccount{
				{
					Login:    "toto",
					Password: "pwd",
				},
			},
			Credentials: []Credential{
				{
					Type:   auth.TLSCertificate,
					Value:  testhelpers.LocalhostCert,
					Value2: testhelpers.LocalhostKey,
				},
			},
		}
		agents := []LocalAgent{updatedAgent}

		require.NoError(t, importLocalAgents(logger, db, agents, false, true))

		var dbAgents model.LocalAgents
		require.NoError(t, db.Select(&dbAgents).OrderBy("id", true).All().Run())
		require.Len(t, dbAgents, 2)
		dbAgent := dbAgents[0]

		t.Run("Check servers", func(t *testing.T) {
			assert.Equal(t, other, dbAgents[1])

			assert.Equal(t, updatedAgent.Name, dbAgent.Name)
			assert.Equal(t, updatedAgent.Protocol, dbAgent.Protocol)
			assert.Equal(t, updatedAgent.Configuration, dbAgent.ProtoConfig.Map())
		})

		t.Run("Check accounts", func(t *testing.T) {
			var accounts model.LocalAccounts
			require.NoError(t, db.Select(&accounts).Where("local_agent_id=?", dbAgent.ID).Run())
			require.Len(t, accounts, 1)
		})

		t.Run("Check credentials", func(t *testing.T) {
			var credentials model.Credentials
			require.NoError(t, db.Select(&credentials).Where("local_agent_id=?", dbAgent.ID).Run())
			assert.Len(t, credentials, 1)

			assert.Equal(t, updatedAgent.Credentials[0].Type, credentials[0].Type)
			assert.Equal(t, updatedAgent.Credentials[0].Value, credentials[0].Value)
			assert.Equal(t, updatedAgent.Credentials[0].Value2, credentials[0].Value2)
		})

		t.Run("Check services", func(t *testing.T) {
			state, exists := services.Servers.State(dbAgent)
			require.True(t, exists)
			assert.Equal(t, utils.StateRunning, state)
			service, _ := services.Servers.Get(dbAgent)
			assert.Equal(t, updatedAgent.Name, service.Name())
		})
	})
}

func TestImportLocalAccounts(t *testing.T) {
	t.Parallel()

	setup := func(tb testing.TB) (*database.DB, *log.Logger, *model.LocalAgent,
		*model.LocalAccount, *model.Credential,
	) {
		tb.Helper()

		db := dbtest.TestDatabase(tb)
		logger := nolog(tb)

		agent := &model.LocalAgent{
			Name: "server", Protocol: testProtocol,
			Address: types.Addr("localhost", 2022),
		}
		require.NoError(t, db.Insert(agent).Run())

		dbAccount := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "foo",
		}
		require.NoError(t, db.Insert(dbAccount).Run())

		accPswd := &model.Credential{
			LocalAccountID: dbAccount.NullableID(),
			Type:           auth.Password,
			Value:          "bar",
		}
		require.NoError(t, db.Insert(accPswd).Run())

		return db, logger, agent, dbAccount, accPswd
	}

	t.Run("New accounts", func(t *testing.T) {
		db, logger, dbAgent, dbAccount, dbPswd := setup(t)

		fileAccount1 := LocalAccount{Login: "toto", Password: "pwd"}
		fileAccount2 := LocalAccount{Login: "tata", Password: "pwd"}
		fileAccounts := []LocalAccount{fileAccount1, fileAccount2}
		require.NoError(t, preprocessLocalAccounts(fileAccounts, dbAgent.Protocol))
		require.NoError(t, importLocalAccounts(logger, db, fileAccounts, dbAgent))

		var accounts model.LocalAccounts
		require.NoError(t, db.Select(&accounts).OrderBy("id", true).Run())
		require.Len(t, accounts, 3)

		t.Run("Check accounts", func(t *testing.T) {
			assert.Equal(t, dbAccount, accounts[0])
			assert.Equal(t, dbAgent.ID, accounts[1].LocalAgentID)
			assert.Equal(t, fileAccount1.Login, accounts[1].Login)
			assert.Equal(t, dbAgent.ID, accounts[1].LocalAgentID)
			assert.Equal(t, fileAccount2.Login, accounts[2].Login)
		})

		t.Run("Check credentials", func(t *testing.T) {
			var creds model.Credentials
			require.NoError(t, db.Select(&creds).OrderBy("local_account_id", true).Run())
			require.Len(t, creds, 3)

			assert.Equal(t, dbPswd, creds[0])

			assert.Equal(t, accounts[1].ID, creds[1].LocalAccountID.Int64)
			assert.Equal(t, auth.Password, creds[1].Type)
			assert.True(t, utils.IsHashOf(creds[1].Value, fileAccount1.Password))

			assert.Equal(t, accounts[2].ID, creds[2].LocalAccountID.Int64)
			assert.Equal(t, auth.Password, creds[2].Type)
			assert.True(t, utils.IsHashOf(creds[2].Value, fileAccount2.Password))
			assert.True(t, utils.IsHashOf(creds[2].Value, fileAccount2.Password))
		})
	})

	t.Run("Update existing", func(t *testing.T) {
		db, logger, dbAgent, dbAccount, dbPswd := setup(t)

		fileAccount := LocalAccount{
			Login: "foo",
			Credentials: []Credential{
				{
					Type:  auth.TLSTrustedCertificate,
					Value: testhelpers.ClientFooCert,
				},
			},
		}
		fileAccounts := []LocalAccount{fileAccount}

		require.NoError(t, importLocalAccounts(logger, db, fileAccounts, dbAgent))

		var accounts model.LocalAccounts
		require.NoError(t, db.Select(&accounts).OrderBy("id", true).Run())
		require.Len(t, accounts, 1)

		assert.Equal(t, accounts[0].Login, dbAccount.Login)

		t.Run("Check credentials", func(t *testing.T) {
			var creds model.Credentials
			require.NoError(t, db.Select(&creds).OrderBy("local_account_id", true).Run())
			require.Len(t, creds, 2)

			assert.Equal(t, dbPswd, creds[0])

			assert.Equal(t, accounts[0].ID, creds[1].LocalAccountID.Int64)
			assert.Equal(t, auth.TLSTrustedCertificate, creds[1].Type)
			assert.Equal(t, fileAccount.Credentials[0].Value, creds[1].Value)
		})
	})
}

func TestR66PasswordImport(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, r66TLS, &r66auth.BcryptAuthHandler{})

	server := &model.LocalAgent{
		Name:     "r66_server",
		Address:  types.Addr("localhost", 0),
		Protocol: r66TLS,
	}
	require.NoError(t, db.Insert(server).Run())

	const pswd = "bar"
	expected := string(r66lib.CryptPass([]byte(pswd)))

	accounts := []LocalAccount{{
		Login:    "foo",
		Password: pswd,
	}}
	require.NoError(t, preprocessLocalAccounts(accounts, server.Protocol))

	require.NoError(t, importLocalAccounts(logger, db, accounts, server))

	var cred model.Credential
	require.NoError(t, db.Get(&cred, "type=?", auth.Password).Run())

	assert.True(t, utils.IsHashOf(cred.Value, expected))
}
