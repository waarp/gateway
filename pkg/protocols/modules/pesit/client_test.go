package pesit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestPesitClientPause(t *testing.T) {
	checkIsPaused := func(tb testing.TB, pip *gwtesting.Pipeline) {
		tb.Helper()

		var servTransfer model.Transfer
		require.NoError(t, pip.Pip.DB.Get(&servTransfer,
			"remote_transfer_id = ? AND local_account_id IS NOT NULL",
			pip.Pip.TransCtx.Transfer.RemoteTransferID).Run(),
			"Failed to retrieve the server transfer")

		assert.Equal(t, types.StatusPaused, servTransfer.Status)
	}

	t.Run("Before data", func(t *testing.T) {
		db := gwtesting.Database(t)
		ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
		cli := newClient(ctx.Client)
		require.NoError(t, cli.Start())

		t.Run("Push", func(t *testing.T) {
			pip := ctx.PushPipeline(t)
			defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)
			requireNoError(t, transfer.Request(), "Failed to connect to partner")

			requireNoError(t, transfer.Pause(), "Failed to pause transfer")
			checkIsPaused(t, &pip)
		})

		t.Run("Pull", func(t *testing.T) {
			pip := ctx.PullPipeline(t)
			defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)
			requireNoError(t, transfer.Request(), "Failed to connect to partner")

			requireNoError(t, transfer.Pause(), "Failed to pause transfer")
			checkIsPaused(t, &pip)
		})
	})

	t.Run("After data", func(t *testing.T) {
		db := gwtesting.Database(t)
		ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
		cli := newClient(ctx.Client)
		require.NoError(t, cli.Start())

		t.Run("Push", func(t *testing.T) {
			pip := ctx.PushPipeline(t)
			defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)
			requireNoError(t, transfer.Request(), "Failed to connect to partner")
			requireNoError(t, transfer.Send(gwtesting.SendFile("hello world")))

			requireNoError(t, transfer.Pause(), "Failed to pause transfer")
			checkIsPaused(t, &pip)
		})

		t.Run("Pull", func(t *testing.T) {
			pip := ctx.PullPipeline(t)
			defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)
			requireNoError(t, transfer.Request(), "Failed to connect to partner")
			requireNoError(t, transfer.Receive(gwtesting.ReceiveFile()))

			requireNoError(t, transfer.Pause(), "Failed to pause transfer")
			checkIsPaused(t, &pip)
		})
	})
}

func TestPesitClientCancel(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	cli := newClient(ctx.Client)

	require.NoError(t, cli.Start())

	t.Run("Given a PESIT pull client", func(t *testing.T) {
		t.Parallel()
		pip := ctx.PullPipeline(t)
		defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

		t.Run("When canceling the transfer", func(t *testing.T) {
			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)

			requireNoError(t, transfer.Request(), "Failed to connect to partner")
			requireNoError(t, transfer.Cancel(), "Failed to cancel the transfer")

			t.Run("Then it should have canceled the server transfer", func(t *testing.T) {
				var servTransfer model.HistoryEntry
				require.NoError(t, db.Get(&servTransfer,
					"remote_transfer_id = ? AND is_server=true",
					pip.Pip.TransCtx.Transfer.RemoteTransferID).Run(),
					"Failed to retrieve the server transfer")

				assert.Equal(t, types.StatusCancelled, servTransfer.Status)
			})
		})
	})

	t.Run("Given a PESIT push client", func(t *testing.T) {
		t.Parallel()
		pip := ctx.PushPipeline(t)
		defer pip.Pip.SetError(types.TeStopped, "transfer stopped")

		t.Run("When canceling the transfer", func(t *testing.T) {
			transfer, err := cli.initTransfer(pip.Pip)
			requireNoError(t, err)

			requireNoError(t, transfer.Request(), "Failed to connect to partner")
			requireNoError(t, transfer.Cancel(), "Failed to cancel the transfer")

			t.Run("Then it should have canceled the server transfer", func(t *testing.T) {
				var servTransfer model.HistoryEntry
				require.NoError(t, db.Get(&servTransfer,
					"remote_transfer_id = ? AND is_server=true",
					pip.Pip.TransCtx.Transfer.RemoteTransferID).Run(),
					"Failed to retrieve the server transfer")

				assert.Equal(t, types.StatusCancelled, servTransfer.Status)
			})
		})
	})
}

func TestClientPreConn(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	cli := newClient(ctx.Client)

	require.NoError(t, cli.Start())

	t.Run("Check credentials", func(t *testing.T) {
		preConnCreds := &model.Credential{
			RemoteAccountID: utils.NewNullInt64(ctx.RemoteAccount.ID),
			Name:            "pre-conn-cred",
			Type:            PreConnectionAuth,
			Value:           "foobar",
			Value2:          "sesame",
		}
		require.NoError(t, db.Insert(preConnCreds).Run())

		pip := ctx.PushPipeline(t)
		trans := pip.Client.(*clientTransfer)

		requireNoError(t, trans.Request())
		t.Cleanup(func() {
			_ = pip.Pip.Cancel(context.Background())
		})

		assert.Equal(t, preConnCreds.Value, trans.client.PreConnectLogin())
		assert.Equal(t, preConnCreds.Value2, trans.client.PreConnectPassword())
	})
}
