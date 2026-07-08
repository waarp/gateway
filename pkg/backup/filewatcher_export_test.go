package backup

import (
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestFileWatcherExport(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	partner := model.RemoteAgent{
		Name:     "test_partner",
		Protocol: testProtocol,
		Address:  types.Addr("127.0.0.1", 2000),
	}
	require.NoError(t, db.Insert(&partner).Run())

	account := model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "test_account",
	}
	require.NoError(t, db.Insert(&account).Run())

	client := model.Client{
		Name:         "test_client",
		Protocol:     testProtocol,
		LocalAddress: types.Addr("127.0.0.1", 3000),
	}
	require.NoError(t, db.Insert(&client).Run())

	rule := model.Rule{
		Name:   "test_rule",
		IsSend: false,
	}
	require.NoError(t, db.Insert(&rule).Run())

	dbFw := &model.FileWatcher{
		Flow:             "test_flow",
		Interval:         30 * time.Second,
		Pattern:          "*.csv",
		NoDuplicateCheck: false,
		RemoteAccount:    account,
		Client:           client,
		Rule:             rule,
	}
	require.NoError(t, db.Insert(dbFw).Run())

	res, err := exportFilewatchers(logger, db)
	require.NoError(t, err)
	require.Len(t, res.Remote, 1)

	assert.Equal(t, file.RemoteFilewatcher{
		Flow:             dbFw.Flow,
		Interval:         file.Duration(dbFw.Interval),
		Pattern:          dbFw.Pattern,
		Partner:          partner.Name,
		RemoteAccount:    account.Login,
		Client:           client.Name,
		Rule:             rule.Name,
		NoDuplicateCheck: dbFw.NoDuplicateCheck,
	}, res.Remote[0])
}
