package gwtesting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const transferTimeout = 10 * time.Minute

var ErrTransferTimedOut = errors.New("transfer timed out")

type Pipeline controller.ClientPipeline

func (p Pipeline) Run() error {
	pip := controller.ClientPipeline(p)
	ctx, cancel := context.WithTimeout(context.Background(), transferTimeout)

	defer cancel()
	// defer pip.Cancel(ctx) //nolint:errcheck //error is unimportant here

	select {
	case err := <-utils.GoRun((&pip).Run):
		return err
	case <-ctx.Done():
		return ErrTransferTimedOut
	}
}

func (ctx *TransferCtx) PushPipeline(tb testing.TB) Pipeline {
	tb.Helper()

	pip, err := controller.NewClientPipeline(ctx.db, ctx.TransferPush)
	require.NoError(tb, err, "Failed to initialize the test push pipeline")

	return Pipeline(*pip)
}

func (ctx *TransferCtx) PullPipeline(tb testing.TB) Pipeline {
	tb.Helper()

	pip, err := controller.NewClientPipeline(ctx.db, ctx.TransferPull)
	require.NoError(tb, err, "Failed to initialize the test pull pipeline")

	return Pipeline(*pip)
}

func (ctx *TransferCtx) RetryPush(tb testing.TB) Pipeline {
	tb.Helper()
	require.NoError(tb, ctx.db.DeleteAll(&model.Task{}).In("rule_id",
		ctx.ClientRulePush.ID, ctx.ServerRulePush.ID).Where("type=?", taskstest.TaskErr).Run())
	ctx.ServerService.SetTracer(func() pipeline.Trace { return pipeline.Trace{} })

	return ctx.PushPipeline(tb)
}

func (ctx *TransferCtx) RetryPull(tb testing.TB) Pipeline {
	tb.Helper()
	require.NoError(tb, ctx.db.DeleteAll(&model.Task{}).In("rule_id",
		ctx.ClientRulePull.ID, ctx.ServerRulePull.ID).Where("type=?", taskstest.TaskErr).Run())
	ctx.ServerService.SetTracer(func() pipeline.Trace { return pipeline.Trace{} })

	return ctx.PullPipeline(tb)
}

func (ctx *TransferCtx) CheckPullTransferOK(tb testing.TB) {
	tb.Helper()
	ctx.checkTransfersOK(tb, ctx.TransferPull, ctx.ServerRulePull)
}

func (ctx *TransferCtx) CheckPushTransferOK(tb testing.TB) {
	tb.Helper()
	ctx.checkTransfersOK(tb, ctx.TransferPush, ctx.ServerRulePush)
}

func (ctx *TransferCtx) checkTransfersOK(tb testing.TB, trans *model.Transfer,
	serverRule *model.Rule,
) {
	tb.Helper()

	var clientTrans model.HistoryEntry

	require.NoError(tb, ctx.db.Get(&clientTrans, "id=?", trans.ID).Run(),
		"Failed retrieve the client history entry")

	ctx.checkTransferOK(tb, &clientTrans)

	var serverTrans model.HistoryEntry

	require.NoError(tb, ctx.db.Get(&serverTrans,
		"is_server=true AND is_send=? AND agent=? AND account=?",
		serverRule.IsSend, ctx.Server.Name, ctx.LocalAccount.Login).Run(),
		"Failed retrieve the server history entry")

	ctx.checkTransferOK(tb, &serverTrans)

	assert.Equal(tb, clientTrans.RemoteTransferID, serverTrans.RemoteTransferID)
}

func (ctx *TransferCtx) checkTransferOK(tb testing.TB, entry *model.HistoryEntry) {
	tb.Helper()
	assert.Equal(tb, types.StatusDone, entry.Status,
		"Then the transfer should be DONE")
	//nolint:testifylint //false positive, the length is the expected value, not the actual
	assert.Equal(tb, int64(len(PullFileContent)), entry.Progress,
		"Then the whole file should have been retrieved")
	//nolint:testifylint //false positive, the length is the expected value, not the actual
	assert.Equal(tb, int64(len(PullFileContent)), entry.Filesize,
		"Then the file size should be correct")
	assert.Zero(tb, entry.ErrCode,
		"Then there shouldn't be any error")
	assert.Zero(tb, entry.ErrDetails,
		"Then there shouldn't be any error message")
}
