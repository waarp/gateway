package tasks

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestTransferPreregister(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	// Setup
	rule := &model.Rule{
		Name:   "send",
		IsSend: true,
	}
	require.NoError(t, db.Insert(rule).Run())

	server := &model.LocalAgent{
		Name:     "test_server",
		Address:  types.Addr("127.0.0.1", 6666),
		Protocol: testProtocol,
	}
	require.NoError(t, db.Insert(server).Run())

	account := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "foobar",
	}
	require.NoError(t, db.Insert(account).Run())

	// Test
	const (
		file     = "test/file.txt"
		validFor = "3d12h"

		oldKey     = "foo"
		oldVal     = "bar"
		updKey     = "fizz"
		origUpdVal = "buzz"
		newUpdVal  = "bazz"
		newKey     = "toto"
		newVal     = "tata"
	)
	dueDate, dErr := tparse.AddDuration(time.Now(), validFor)
	require.NoError(t, dErr)

	runner := TransferPreregister{}
	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{
			TransferInfo: map[string]any{
				oldKey: oldVal,
				updKey: origUpdVal,
			},
		},
	}

	require.NoError(t, runner.Run(t.Context(), map[string]string{
		"file":     file,
		"rule":     rule.Name,
		"isSend":   strconv.FormatBool(rule.IsSend),
		"server":   server.Name,
		"account":  account.Login,
		"validFor": validFor,
		"copyInfo": "true",
		"info": fmt.Sprintf(`{"%s": "%s", "%s": "%s"}`,
			updKey, newUpdVal, newKey, newVal),
	}, db, logger, transCtx, nil))

	var check model.Transfer
	require.NoError(t, db.Get(&check, "rule_id=?", rule.ID).Run())

	if rule.IsSend {
		assert.Equal(t, file, check.SrcFilename)
	} else {
		assert.Equal(t, file, check.DestFilename)
	}

	assert.Equal(t, types.StatusAvailable, check.Status)
	assert.Equal(t, rule.ID, check.RuleID)
	assert.Equal(t, account.ID, check.LocalAccountID.Int64)
	assert.WithinDuration(t, dueDate, check.Start, time.Second)
	assert.Subset(t, check.TransferInfo, map[string]any{
		oldKey: oldVal,
		updKey: newUpdVal,
		newKey: newVal,
	})
}
