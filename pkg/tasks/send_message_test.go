package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestMessage(t *testing.T) {
	t.Parallel()

	logger := logtest.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	partner := &model.RemoteAgent{
		Name:     "test_partner",
		Protocol: testProtocol,
		Address:  types.Addr("1.2.3.4", 5),
	}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{
		RemoteAgent: *partner,
		Login:       "test_account",
	}
	require.NoError(t, db.Insert(account).Run())

	client := &model.Client{Name: "test_client", Protocol: testProtocol}
	require.NoError(t, db.Insert(client).Run())

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{RemoteTransferID: "123456"},
	}
	remote := &testMessageRemote{}
	task := &sendMessageTask{}
	args := map[string]string{
		"client":  client.Name,
		"partner": partner.Name,
		"account": account.Login,
		"message": "hello world!",
	}

	require.NoError(t, task.Run(t.Context(), args, db, logger, transCtx, remote))

	assert.Equal(t, client, remote.client)
	assert.Equal(t, partner, remote.partner)
	assert.Equal(t, account, remote.account)
	assert.Equal(t, transCtx.Transfer.RemoteTransferID, remote.transferID)
	assert.Equal(t, "hello world!", remote.message)
}

type testMessageRemote struct {
	client     *model.Client
	partner    *model.RemoteAgent
	account    *model.RemoteAccount
	transferID string
	message    string
}

func (t *testMessageRemote) SendMessage(_ *database.DB, _ *log.Logger,
	client *model.Client, partner *model.RemoteAgent, account *model.RemoteAccount,
	transferID, message string,
) error {
	t.client = client
	t.partner = partner
	t.account = account
	t.transferID = transferID
	t.message = message

	return nil
}
