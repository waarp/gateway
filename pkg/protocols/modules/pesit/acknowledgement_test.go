package pesit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestWaitAckClient(t *testing.T) {
	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, map[string]any{
		"expectsAck": true,
	})

	pip := ctx.PushPipeline(t)
	require.NoError(t, pip.Run())

	var check model.NormalizedTransferView
	require.NoError(t, db.Get(&check, "id=?", ctx.TransferPush.ID).Eager().Run())

	assert.True(t, check.IsTransfer)
	assert.Equal(t, types.StatusRunning, check.Status)
	assert.Subset(t, check.TransferInfo, map[string]any{ackExpectedKey: true})
}

func TestWaitAckServer(t *testing.T) {
	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	serverTransfer := &model.Transfer{
		RuleID:         ctx.ServerRulePull.ID,
		LocalAccountID: ctx.LocalAccount.NullableID(),
		SrcFilename:    ctx.TransferPull.SrcFilename,
		Start:          time.Now().Add(time.Hour),
		Status:         types.StatusAvailable,
		TransferInfo:   map[string]any{ackExpectedKey: true},
	}
	require.NoError(t, db.Insert(serverTransfer).Run())

	pip := ctx.PullPipeline(t)
	require.NoError(t, pip.Run())

	var check model.NormalizedTransferView
	require.NoError(t, db.Get(&check, "id=?", serverTransfer.ID).Eager().Run())

	assert.True(t, check.IsTransfer)
	assert.Equal(t, types.StatusRunning, check.Status)
	assert.Subset(t, check.TransferInfo, map[string]any{ackExpectedKey: true})
}
