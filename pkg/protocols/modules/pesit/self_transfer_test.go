package pesit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestOk(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer", func(t *testing.T) {
		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push client", func(t *testing.T) {
		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorPreTasksClient(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a client pre-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ClientRulePull, model.ChainPre)

		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a client pre-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ClientRulePush, model.ChainPre)

		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorPreTasksServer(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a server pre-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ServerRulePull, model.ChainPre)

		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a server pre-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ServerRulePush, model.ChainPre)

		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorPostTasksClient(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a client post-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ClientRulePull, model.ChainPost)

		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a client post-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ClientRulePush, model.ChainPost)

		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorPostTasksServer(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a server post-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ServerRulePull, model.ChainPost)

		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a server post-task error", func(t *testing.T) {
		ctx.AddTaskError(t, ctx.ServerRulePush, model.ChainPost)

		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorDataClient(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a client data error", func(t *testing.T) {
		pip := ctx.PullPipeline(t)
		gwtesting.AddClientDataError(t, &pip)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a client data error", func(t *testing.T) {
		pip := ctx.PushPipeline(t)
		gwtesting.AddClientDataError(t, &pip)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestErrorDataServer(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)

	t.Run("Given a PESIT pull transfer with a server data error", func(t *testing.T) {
		pip := ctx.PullPipeline(t)
		ctx.AddServerDataError(t, ctx.ServerRulePull)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPull(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push transfer with a server data error", func(t *testing.T) {
		pip := ctx.PushPipeline(t)
		ctx.AddServerDataError(t, ctx.ServerRulePush)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.Error(t, pip.Run(), "Then the transfer should fail")

			newPip := ctx.RetryPush(t)
			require.NoError(t, newPip.Run(), "Then the new transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}

func TestCFT(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit,
		&ServerConfig{CompatibilityMode: CompatibilityModeNonStandard},
		nil,
		&PartnerConfig{CompatibilityMode: CompatibilityModeNonStandard},
	)

	t.Run("Given a PESIT pull transfer", func(t *testing.T) {
		serverPullTrans := &model.Transfer{
			RuleID:         ctx.ServerRulePull.ID,
			LocalAccountID: ctx.LocalAccount.GetNullID(),
			SrcFilename:    ctx.TransferPull.SrcFilename,
			Start:          time.Date(9999, 1, 1, 1, 0, 0, 0, time.UTC),
			Status:         types.StatusAvailable,
		}
		require.NoError(t, db.Insert(serverPullTrans).Run())

		servFreetext := map[string]any{serverTransFreetextKey: "pesit freetext sample"}
		require.NoError(t, serverPullTrans.SetTransferInfo(db, servFreetext))

		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)

				hist := &model.HistoryEntry{ID: ctx.TransferPull.ID}
				infoCheck, infoErr := hist.GetTransferInfo(db)
				require.NoError(t, infoErr)
				assert.Subset(t, infoCheck, servFreetext)
			})
		})
	})

	t.Run("Given a PESIT push client", func(t *testing.T) {
		serverPushTrans := &model.Transfer{
			RuleID:         ctx.ServerRulePush.ID,
			LocalAccountID: ctx.LocalAccount.GetNullID(),
			DestFilename:   ctx.TransferPull.DestFilename,
			Start:          time.Date(9999, 1, 1, 1, 0, 0, 0, time.UTC),
			Status:         types.StatusAvailable,
			Filesize:       model.UnknownSize,
		}
		require.NoError(t, db.Insert(serverPushTrans).Run())

		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}
