package tasks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type testClientPipeline struct {
	done bool
	err  error
}

func (t *testClientPipeline) Run() error {
	t.done = true

	return t.err
}

func (t *testClientPipeline) Interrupt(context.Context) error { return nil }

func TestTransferRun(t *testing.T) {
	replaceArg := func(tb testing.TB, args map[string]string, key, value string) {
		oldVal := args[key]
		args[key] = value
		tb.Cleanup(func() { args[key] = oldVal })
	}

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	push := &model.Rule{
		Name:   "transfer rule",
		IsSend: true,
		Path:   "rule/path",
	}
	require.NoError(t, db.Insert(push).Run())

	pull := &model.Rule{
		Name:   "transfer rule",
		IsSend: false,
		Path:   "rule/path",
	}
	require.NoError(t, db.Insert(pull).Run())

	client := &model.Client{Name: "cli", Protocol: testProtocol}
	require.NoError(t, db.Insert(client).Run())

	GetDefaultTransferClient = func(database.Access, string) (*model.Client, error) {
		return client, nil
	}

	partner := &model.RemoteAgent{
		Name: "test partner", Protocol: client.Protocol,
		Address: types.Addr("localhost", 1111),
	}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "test account",
	}
	require.NoError(t, db.Insert(account).Run())

	transCtx := func(tb testing.TB) *model.TransferContext {
		parentTransfer := &model.Transfer{
			RemoteAccountID: utils.NewNullInt64(account.ID),
			ClientID:        utils.NewNullInt64(client.ID),
			RuleID:          pull.ID,
			SrcFilename:     "/old/test/file",
			TransferInfo:    map[string]any{"foo": "bar", "baz": true},
			TaskNumber:      2,
		}
		require.NoError(t, db.Insert(parentTransfer).Run())

		return &model.TransferContext{
			Transfer:      parentTransfer,
			Rule:          pull,
			RemoteAgent:   partner,
			RemoteAccount: account,
		}
	}

	t.Run("Send transfer", func(t *testing.T) {
		const (
			filename = "/send/test/file.src"
			output   = "/send/test/file.dst"
			copyInfo = "true"
			info     = `{"baz": "qux", "real": true, "delay": 10, "__followID__": 12345}`
		)

		runner := &TransferTask{}
		args := map[string]string{
			"file":                 filename,
			"output":               output,
			"using":                client.Name,
			"to":                   partner.Name,
			"as":                   account.Login,
			"rule":                 push.Name,
			"copyInfo":             copyInfo,
			"info":                 info,
			"nbOfAttempts":         "5",
			"firstRetryDelay":      "1m30s",
			"retryIncrementFactor": "1.5",
		}

		t.Run("Valid task", func(t *testing.T) {
			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			require.NoError(t, err)

			var transfer model.Transfer
			require.NoError(t, db.Get(&transfer, "src_filename=?", filename).Run())

			assert.Equal(t, output, transfer.DestFilename)
			assert.Equal(t, account.ID, transfer.RemoteAccountID.Int64)
			assert.Equal(t, push.ID, transfer.RuleID)
			assert.Equal(t, client.ID, transfer.ClientID.Int64)
			assert.EqualValues(t, 5, transfer.RemainingTries)
			assert.EqualValues(t, 90, transfer.NextRetryDelay)
			assert.EqualValues(t, 1.5, transfer.RetryIncrementFactor)
			assert.Equal(t, map[string]any{
				"foo":          "bar",
				"baz":          "qux",
				"real":         true,
				"delay":        json.Number("10"),
				model.FollowID: json.Number("12345"),
			}, transfer.TransferInfo)
		})

		t.Run("Partner does not exist", func(t *testing.T) {
			replaceArg(t, args, "to", "toto")

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			assert.ErrorIs(t, err, ErrTransferPartnerNotFound)
		})

		t.Run("Account does not exist", func(t *testing.T) {
			replaceArg(t, args, "as", "toto")

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			assert.ErrorIs(t, err, ErrTransferAccountNotFound)
		})

		t.Run("Rule does not exist", func(t *testing.T) {
			replaceArg(t, args, "rule", "toto")

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			assert.ErrorIs(t, err, ErrTransferRuleNotFound)
		})

		t.Run("Client does not exist", func(t *testing.T) {
			replaceArg(t, args, "using", "toto")

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			assert.ErrorIs(t, err, ErrTransferClientNotFound)
		})

		t.Run("With default client", func(t *testing.T) {
			delete(args, "using")

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			require.NoError(t, err)
		})

		t.Run("Synchronous transfer", func(t *testing.T) {
			args["synchronous"] = "true"

			var check testClientPipeline
			NewClientPipeline = func(db *database.DB, trans *model.Transfer) (ClientPipeline, error) {
				return &check, nil
			}

			err := runner.Run(t.Context(), args, db, logger, transCtx(t), nil)
			require.NoError(t, err)

			assert.True(t, check.done)
		})

		t.Run("Synchronous with resume", func(t *testing.T) {
			args["synchronous"] = "true"

			var transID int64
			ctx := transCtx(t)

			check := testClientPipeline{err: assert.AnError}
			NewClientPipeline = func(db *database.DB, trans *model.Transfer) (ClientPipeline, error) {
				transID = trans.ID

				return &check, nil
			}

			err := runner.Run(t.Context(), args, db, logger, ctx, nil)
			require.ErrorIs(t, err, assert.AnError)

			assert.True(t, check.done)

			check2 := testClientPipeline{}
			NewClientPipeline = func(db *database.DB, trans *model.Transfer) (ClientPipeline, error) {
				require.Equal(t, transID, trans.ID)

				return &check2, nil
			}

			err = runner.Run(t.Context(), args, db, logger, ctx, nil)
			require.NoError(t, err)

			assert.True(t, check2.done)
		})
	})

	t.Run("Receive transfer", func(t *testing.T) {
		const filename = "/recv/test/file.txt"

		trans := &TransferTask{}
		args := map[string]string{
			"file":  filename,
			"using": client.Name,
			"from":  partner.Name,
			"as":    account.Login,
			"rule":  pull.Name,
		}

		t.Run("Valid task", func(t *testing.T) {
			err := trans.Run(t.Context(), args, db, logger, transCtx(t), nil)
			require.NoError(t, err)

			var transfer model.Transfer
			require.NoError(t, db.Get(&transfer, "src_filename=?", filename).Run())

			assert.Equal(t, account.ID, transfer.RemoteAccountID.Int64)
			assert.Equal(t, pull.ID, transfer.RuleID)
			assert.Equal(t, client.ID, transfer.ClientID.Int64)
		})

		t.Run("Partner does not exist", func(t *testing.T) {
			replaceArg(t, args, "from", "toto")

			err := trans.Run(t.Context(), args, db, logger, transCtx(t), nil)
			assert.ErrorIs(t, err, ErrTransferPartnerNotFound)
		})
	})
}
