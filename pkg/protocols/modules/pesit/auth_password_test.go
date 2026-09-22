package pesit

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthPassword(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	credentialExists := func(tb testing.TB, sql string, args ...any) bool {
		n, err := db.Count(&model.Credential{}).Where(sql, args...).Run()
		require.NoError(tb, err)

		return n > 0
	}

	t.Run("Server", func(t *testing.T) {
		t.Parallel()

		server := &model.LocalAgent{
			Name:     "test_server",
			Address:  types.Addr("", 123),
			Protocol: Pesit,
		}
		require.NoError(t, db.Insert(server).Run())

		servPreConn := &model.Credential{
			LocalAgentID: server.NullableID(),
			Type:         PreConnectionAuth,
			Value:        "toto",
			Value2:       "sesame",
		}
		require.Error(t, db.Insert(servPreConn).Run(),
			"pre-conn credentials on servers is NOT allowed")
		assert.False(t, credentialExists(t, "local_agent_id=?", server.ID))

		t.Run("Local account", func(t *testing.T) {
			t.Parallel()

			account := &model.LocalAccount{
				LocalAgent: *server,
				Login:      "test_account",
			}
			require.NoError(t, db.Insert(account).Run())

			accPreConn := &model.Credential{
				LocalAccountID: account.NullableID(),
				Type:           PreConnectionAuth,
				Value:          "toto",
				Value2:         "sesame",
			}
			require.Error(t, db.Insert(accPreConn).Run(),
				"pre-conn credentials on local accounts is NOT allowed")
			assert.False(t, credentialExists(t, "local_account_id=?", account.ID))
		})
	})

	t.Run("Partner", func(t *testing.T) {
		t.Parallel()

		partner := &model.RemoteAgent{
			Name:     "test_partner",
			Address:  types.Addr("", 1234),
			Protocol: Pesit,
		}
		require.NoError(t, db.Insert(partner).Run())

		servPreConn := &model.Credential{
			RemoteAgentID: partner.NullableID(),
			Type:          PreConnectionAuth,
			Value:         "toto",
			Value2:        "sesame",
		}
		require.Error(t, db.Insert(servPreConn).Run(),
			"pre-conn credentials on partners is NOT allowed")
		assert.False(t, credentialExists(t, "remote_agent_id=?", partner.ID))

		t.Run("Remote account", func(t *testing.T) {
			t.Parallel()

			account := &model.RemoteAccount{
				RemoteAgent: *partner,
				Login:       "test_account",
			}
			require.NoError(t, db.Insert(account).Run())

			accPreConn := &model.Credential{
				RemoteAccountID: account.NullableID(),
				Type:            PreConnectionAuth,
				Value:           "toto",
				Value2:          "sesame",
			}
			require.NoError(t, db.Insert(accPreConn).Run(),
				"pre-conn credentials on remote accounts is permitted")
			assert.True(t, credentialExists(t, "remote_account_id=?", account.ID))
		})
	})
}
