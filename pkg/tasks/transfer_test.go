package tasks

import (
	"context"
	"fmt"
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

	GetDefaultTransferClient = func(*database.DB, string) (*model.Client, error) {
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

	oldTransfer := &model.Transfer{
		RemoteAccountID: utils.NewNullInt64(account.ID),
		ClientID:        utils.NewNullInt64(client.ID),
		RuleID:          pull.ID,
		SrcFilename:     "/old/test/file",
	}
	require.NoError(t, db.Insert(oldTransfer).Run())

	transCtx := &model.TransferContext{
		Transfer: oldTransfer,
		TransInfo: map[string]any{
			"foo": "bar", "baz": true,
		},
		Rule:          pull,
		RemoteAgent:   partner,
		RemoteAccount: account,
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
			err := runner.Run(context.Background(), args, db, logger, transCtx)
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

			transInfo, infoErr := transfer.GetTransferInfo(db)
			require.NoError(t, infoErr)
			assert.JSONEq(t,
				fmt.Sprintf(`{
					"foo":          "bar",
					"baz":          "qux",
					"real":         true,
					"delay":        10,
					%q: 12345
				}`, model.FollowID),
				testhelpers.MustMarshalJSON(t, transInfo),
			)
		})

		t.Run("Partner does not exist", func(t *testing.T) {
			replaceArg(t, args, "to", "toto")

			err := runner.Run(context.Background(), args, db, logger, transCtx)
			assert.ErrorIs(t, err, ErrTransferPartnerNotFound)
		})

		t.Run("Account does not exist", func(t *testing.T) {
			replaceArg(t, args, "as", "toto")

			err := runner.Run(context.Background(), args, db, logger, transCtx)
			assert.ErrorIs(t, err, ErrTransferAccountNotFound)
		})

		t.Run("Rule does not exist", func(t *testing.T) {
			replaceArg(t, args, "rule", "toto")

			err := runner.Run(context.Background(), args, db, logger, transCtx)
			assert.ErrorIs(t, err, ErrTransferRuleNotFound)
		})

		t.Run("Client does not exist", func(t *testing.T) {
			replaceArg(t, args, "using", "toto")

			err := runner.Run(context.Background(), args, db, logger, transCtx)
			assert.ErrorIs(t, err, ErrTransferClientNotFound)
		})

		t.Run("With default client", func(t *testing.T) {
			delete(args, "using")

			err := runner.Run(context.Background(), args, db, logger, transCtx)
			require.NoError(t, err)
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
			err := trans.Run(context.Background(), args, db, logger, transCtx)
			require.NoError(t, err)

			var transfer model.Transfer
			require.NoError(t, db.Get(&transfer, "src_filename=?", filename).Run())

			assert.Equal(t, account.ID, transfer.RemoteAccountID.Int64)
			assert.Equal(t, pull.ID, transfer.RuleID)
			assert.Equal(t, client.ID, transfer.ClientID.Int64)
		})

		t.Run("Partner does not exist", func(t *testing.T) {
			replaceArg(t, args, "from", "toto")

			err := trans.Run(context.Background(), args, db, logger, transCtx)
			assert.ErrorIs(t, err, ErrTransferPartnerNotFound)
		})
	})
}
