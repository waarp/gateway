package backup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestFilewatcherImport(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)

	const (
		partnerName = "test_partner"
		accountName = "test_account"
		clientName  = "test_client"
		ruleName    = "test_rule"
		fwFlow      = "test_flow"
	)

	setup := func(tb testing.TB) (*database.DB, *model.RemoteAgent,
		*model.RemoteAccount, *model.Client, *model.Rule, *model.FileWatcher,
	) {
		tb.Helper()
		db := dbtest.TestDatabase(tb)

		partner := model.RemoteAgent{
			Name:     partnerName,
			Protocol: testProtocol,
			Address:  types.Addr("127.0.0.1", 2000),
		}
		require.NoError(tb, db.Insert(&partner).Run())

		account := model.RemoteAccount{
			RemoteAgentID: partner.ID,
			Login:         accountName,
		}
		require.NoError(tb, db.Insert(&account).Run())

		client := model.Client{
			Name:         clientName,
			Protocol:     testProtocol,
			LocalAddress: types.Addr("127.0.0.1", 3000),
		}
		require.NoError(tb, db.Insert(&client).Run())

		rule := model.Rule{
			Name:   ruleName,
			IsSend: false,
		}
		require.NoError(tb, db.Insert(&rule).Run())

		dbFw := model.FileWatcher{
			Flow:             fwFlow,
			Interval:         30 * time.Second,
			Pattern:          "*.csv",
			NoDuplicateCheck: false,
			RemoteAccount:    account,
			Client:           client,
			Rule:             rule,
		}
		require.NoError(tb, db.Insert(&dbFw).Run())

		return db, &partner, &account, &client, &rule, &dbFw
	}

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		remFw1 := file.RemoteFilewatcher{
			Flow:             "new_flow",
			Interval:         file.Duration(time.Minute),
			Pattern:          "*.txt",
			Partner:          partnerName,
			RemoteAccount:    accountName,
			Client:           clientName,
			Rule:             ruleName,
			NoDuplicateCheck: true,
		}
		input := &file.FileWatchers{
			Remote: []file.RemoteFilewatcher{remFw1},
		}

		t.Run("With reset", func(t *testing.T) {
			t.Parallel()
			db, _, account, client, rule, _ := setup(t)

			require.NoError(t, importFilewatchers(logger, db, input, true, false))

			var fws model.FileWatchers
			require.NoError(t, db.Select(&fws).Eager().Run())
			require.Len(t, fws, 1)

			assert.Equal(t, remFw1.Flow, fws[0].Flow)
			assert.Equal(t, time.Duration(remFw1.Interval), fws[0].Interval)
			assert.Equal(t, remFw1.Pattern, fws[0].Pattern)
			assert.Equal(t, account, &fws[0].RemoteAccount)
			assert.Equal(t, client, &fws[0].Client)
			assert.Equal(t, rule, &fws[0].Rule)
			assert.Equal(t, remFw1.NoDuplicateCheck, fws[0].NoDuplicateCheck)
		})

		t.Run("No reset", func(t *testing.T) {
			t.Parallel()
			db, _, account, client, rule, existing := setup(t)

			require.NoError(t, importFilewatchers(logger, db, input, false, false))

			var fws model.FileWatchers
			require.NoError(t, db.Select(&fws).Eager().OrderBy("id", true).Run())
			require.Len(t, fws, 2)

			assert.Equal(t, existing, fws[0])

			assert.Equal(t, remFw1.Flow, fws[1].Flow)
			assert.Equal(t, time.Duration(remFw1.Interval), fws[1].Interval)
			assert.Equal(t, remFw1.Pattern, fws[1].Pattern)
			assert.Equal(t, account, &fws[1].RemoteAccount)
			assert.Equal(t, client, &fws[1].Client)
			assert.Equal(t, rule, &fws[1].Rule)
			assert.Equal(t, remFw1.NoDuplicateCheck, fws[1].NoDuplicateCheck)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		remFw1 := file.RemoteFilewatcher{
			Flow:             fwFlow,
			Interval:         file.Duration(time.Minute),
			Pattern:          "*.txt",
			Partner:          partnerName,
			RemoteAccount:    accountName,
			Client:           clientName,
			Rule:             ruleName,
			NoDuplicateCheck: true,
		}
		input := &file.FileWatchers{
			Remote: []file.RemoteFilewatcher{remFw1},
		}

		db, _, account, client, rule, _ := setup(t)

		require.NoError(t, importFilewatchers(logger, db, input, true, false))

		var fws model.FileWatchers
		require.NoError(t, db.Select(&fws).Eager().Run())
		require.Len(t, fws, 1)

		assert.Equal(t, remFw1.Flow, fws[0].Flow)
		assert.Equal(t, time.Duration(remFw1.Interval), fws[0].Interval)
		assert.Equal(t, remFw1.Pattern, fws[0].Pattern)
		assert.Equal(t, account, &fws[0].RemoteAccount)
		assert.Equal(t, client, &fws[0].Client)
		assert.Equal(t, rule, &fws[0].Rule)
		assert.Equal(t, remFw1.NoDuplicateCheck, fws[0].NoDuplicateCheck)
	})
}
