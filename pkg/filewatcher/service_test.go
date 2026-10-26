package filewatcher

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestFilewatcherService(t *testing.T) {
	t.Parallel()

	models := makeTestFTPModels(t)
	ctx := makeTestContext(t, models)

	client := &model.Client{
		Name:     models.partner.Protocol + "-client",
		Protocol: models.partner.Protocol,
	}
	require.NoError(t, ctx.db.Insert(client).Run())

	rule := &model.Rule{
		Name:   "default",
		IsSend: false,
	}
	require.NoError(t, ctx.db.Insert(rule).Run())

	dbFw := &model.FileWatcher{
		Flow:            "test",
		Interval:        0,
		Pattern:         "*",
		RemoteAccountID: models.user.ID,
		ClientID:        client.ID,
		RuleID:          rule.ID,
	}
	require.NoError(t, ctx.db.Insert(dbFw).Run())

	fw := NewFilewatcher(ctx.db, dbFw)
	require.NoError(t, fw.Start())
	defer fw.Stop(t.Context())

	require.NoError(t, fw.FireOnce(fw.fw))
}
