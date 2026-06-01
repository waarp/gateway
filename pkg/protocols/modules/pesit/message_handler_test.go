package pesit

import (
	"net"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestMessageHandlerClient(t *testing.T) {
	t.Parallel()

	// ############ SETUP CONFIG ###############
	db := dbtest.TestDatabase(t)

	const (
		origPartnerName = "origin_partner"
		destPartnerName = "dest_partner"
		password        = "sesame"

		followID = 123456
		message  = "transfer acknowledged"
	)

	// ############ SETUP LOCAL SERVER ###############
	server := &model.LocalAgent{Name: "server", Address: gwtesting.Address(t), Protocol: Pesit}
	require.NoError(t, db.Insert(server).Run())
	locAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: destPartnerName}
	require.NoError(t, db.Insert(locAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: locAccount.NullableID(),
		Type:           auth.Password, Value: password,
	}).Run())

	// ############ SETUP ORIGIN PARTNER ###############
	origPartnerServer := makeTestMessageServer(t)
	origPartner := &model.RemoteAgent{
		Name:     origPartnerName,
		Protocol: Pesit,
		Address:  types.MustAddr(origPartnerServer.addr),
	}
	require.NoError(t, db.Insert(origPartner).Run())
	origAccount := &model.RemoteAccount{RemoteAgentID: origPartner.ID, Login: server.Name}
	require.NoError(t, db.Insert(origAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		RemoteAccountID: origAccount.NullableID(),
		Type:            auth.Password, Value: password,
	}).Run())

	// ############ SETUP DEST PARTNER ###############
	destPartner := &model.RemoteAgent{
		Name:     destPartnerName,
		Protocol: Pesit,
		Address:  types.Addr("localhost", 1234),
	}
	require.NoError(t, db.Insert(destPartner).Run())
	destAccount := &model.RemoteAccount{RemoteAgentID: destPartner.ID, Login: server.Name}
	require.NoError(t, db.Insert(destAccount).Run())

	// ############ SETUP LOCAL CLIENT ###############
	client := &model.Client{Name: "client", Protocol: Pesit}
	require.NoError(t, db.Insert(client).Run())

	// ############ SETUP RULES ###############
	ruleRecv := &model.Rule{Name: "recv", IsSend: false}
	ruleSend := &model.Rule{Name: "push", IsSend: true}
	require.NoError(t, db.Insert(ruleRecv).Run())
	require.NoError(t, db.Insert(ruleSend).Run())

	// ############ SETUP TRANSFERS ###############
	transOrig := &model.Transfer{
		Status:          types.StatusRunning,
		ClientID:        client.NullableID(),
		RuleID:          ruleRecv.ID,
		RemoteAccountID: origAccount.NullableID(),
		SrcFilename:     "test.txt",
		TransferInfo:    map[string]any{model.FollowID: followID},
	}
	require.NoError(t, db.Insert(transOrig).Run())
	transDest := &model.Transfer{
		Status:          types.StatusRunning,
		ClientID:        client.NullableID(),
		RuleID:          ruleSend.ID,
		RemoteAccountID: destAccount.NullableID(),
		SrcFilename:     "test.txt",
		TransferInfo:    map[string]any{model.FollowID: followID},
	}
	require.NoError(t, db.Insert(transDest).Run())

	// ############ START SERVICE ###############
	serv := newService(db, server)
	require.NoError(t, serv.Start())

	// ############ CONNECT ###############
	pesitClient := pesit.NewClient(locAccount.Login, password, server.Name)
	pesitClient.SetNSDUUsage(true)
	conn, err := net.Dial("tcp", server.Address.String())
	require.NoError(t, err)
	require.NoError(t, pesitClient.Connect(conn))

	// ############ SEND MESSAGE ###############
	transferID, err := utils.ParseUint[uint32](transDest.RemoteTransferID)
	require.NoError(t, err)
	require.NoError(t, pesitClient.SendMessage(transferID, message))

	// ############ CHECK ORIGIN ###############
	assert.Equal(t, origAccount.Login, origPartnerServer.login)
	assert.Equal(t, password, origPartnerServer.pswd)
	assert.Equal(t, transOrig.RemoteTransferID, utils.FormatUint(origPartnerServer.msg.TransferID))
	assert.Equal(t, message, origPartnerServer.msg.Message)

	// ############ CHECK TRANSFERS ###############
	var check model.NormalizedTransfers
	require.NoError(t, db.Select(&check).OrderBy("id", true).Eager().Run())
	require.Len(t, check, 2)

	assert.Equal(t, transOrig.ID, check[0].ID)
	assert.False(t, check[0].IsTransfer)
	assert.Equal(t, types.StatusDone, check[0].Status)
	assert.Subset(t, check[0].TransferInfo, map[string]any{
		ackSentKey: true,
	})

	assert.Equal(t, transDest.ID, check[1].ID)
	assert.False(t, check[1].IsTransfer)
	assert.Equal(t, types.StatusDone, check[1].Status)
	assert.Subset(t, check[1].TransferInfo, map[string]any{
		ackReceivedKey: true,
	})
}

func TestMessageHandlerServer(t *testing.T) {
	t.Parallel()

	// ############ SETUP CONFIG ###############
	db := dbtest.TestDatabase(t)

	const (
		origPartnerName = "origin_partner"
		destPartnerName = "dest_partner"
		password        = "sesame"

		followID = 123456
		message  = "transfer acknowledged"
	)

	// ############ SETUP LOCAL SERVER ###############
	server := &model.LocalAgent{Name: "server", Address: gwtesting.Address(t), Protocol: Pesit}
	require.NoError(t, db.Insert(server).Run())
	destLocAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: destPartnerName}
	require.NoError(t, db.Insert(destLocAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: destLocAccount.NullableID(),
		Type:           auth.Password,
		Value:          password,
	}).Run())
	origLocAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: origPartnerName}
	require.NoError(t, db.Insert(origLocAccount).Run())

	// ############ SETUP ORIGIN PARTNER ###############
	origPartnerServer := makeTestMessageServer(t)
	origPartner := &model.RemoteAgent{
		Name:     origPartnerName,
		Protocol: Pesit,
		Address:  types.MustAddr(origPartnerServer.addr),
	}
	require.NoError(t, db.Insert(origPartner).Run())
	origAccount := &model.RemoteAccount{RemoteAgentID: origPartner.ID, Login: server.Name}
	require.NoError(t, db.Insert(origAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		RemoteAccountID: origAccount.NullableID(),
		Type:            auth.Password, Value: password,
	}).Run())

	// ############ SETUP DEST PARTNER ###############
	destPartner := &model.RemoteAgent{
		Name:     destPartnerName,
		Protocol: Pesit,
		Address:  types.Addr("localhost", 1234),
	}
	require.NoError(t, db.Insert(destPartner).Run())
	destAccount := &model.RemoteAccount{RemoteAgentID: destPartner.ID, Login: server.Name}
	require.NoError(t, db.Insert(destAccount).Run())

	// ############ SETUP LOCAL CLIENT ###############
	client := &model.Client{Name: "client", Protocol: Pesit}
	require.NoError(t, db.Insert(client).Run())

	// ############ SETUP RULES ###############
	ruleRecv := &model.Rule{Name: "recv", IsSend: false}
	ruleSend := &model.Rule{Name: "push", IsSend: true}
	require.NoError(t, db.Insert(ruleRecv).Run())
	require.NoError(t, db.Insert(ruleSend).Run())

	// ############ SETUP TRANSFERS ###############
	transOrig := &model.Transfer{
		Status:         types.StatusRunning,
		RuleID:         ruleRecv.ID,
		LocalAccountID: origLocAccount.NullableID(),
		DestFilename:   "test.txt",
		TransferInfo:   map[string]any{model.FollowID: followID},
	}
	require.NoError(t, db.Insert(transOrig).Run())
	transDest := &model.Transfer{
		Status:          types.StatusRunning,
		ClientID:        client.NullableID(),
		RuleID:          ruleSend.ID,
		RemoteAccountID: destAccount.NullableID(),
		SrcFilename:     "test.txt",
		TransferInfo:    map[string]any{model.FollowID: followID},
	}
	require.NoError(t, db.Insert(transDest).Run())

	// ############ START SERVICE ###############
	serv := newService(db, server)
	require.NoError(t, serv.Start())

	// ############ CONNECT ###############
	pesitClient := pesit.NewClient(destLocAccount.Login, password, server.Name)
	pesitClient.SetNSDUUsage(true)
	conn, err := net.Dial("tcp", server.Address.String())
	require.NoError(t, err)
	require.NoError(t, pesitClient.Connect(conn))

	// ############ SEND MESSAGE ###############
	transferID, err := utils.ParseUint[uint32](transDest.RemoteTransferID)
	require.NoError(t, err)
	require.NoError(t, pesitClient.SendMessage(transferID, message))

	// ############ CHECK ORIGIN ###############
	assert.Equal(t, origAccount.Login, origPartnerServer.login)
	assert.Equal(t, password, origPartnerServer.pswd)
	assert.Equal(t, transOrig.RemoteTransferID, utils.FormatUint(origPartnerServer.msg.TransferID))
	assert.Equal(t, message, origPartnerServer.msg.Message)

	// ############ CHECK TRANSFERS ###############
	var check model.NormalizedTransfers
	require.NoError(t, db.Select(&check).OrderBy("id", true).Eager().Run())
	require.Len(t, check, 2)

	assert.Equal(t, transOrig.ID, check[0].ID)
	assert.False(t, check[0].IsTransfer)
	assert.Equal(t, types.StatusDone, check[0].Status)
	assert.Subset(t, check[0].TransferInfo, map[string]any{
		ackSentKey: true,
	})

	assert.Equal(t, transDest.ID, check[1].ID)
	assert.False(t, check[1].IsTransfer)
	assert.Equal(t, types.StatusDone, check[1].Status)
	assert.Subset(t, check[1].TransferInfo, map[string]any{
		ackReceivedKey: true,
	})
}
