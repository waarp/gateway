package pesit

import (
	"io"
	"net"
	"testing"
	"time"

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
		origPartnerName  = "origin_partner"
		origPartnerLogin = "origin_partner_login"
		destPartnerName  = "dest_partner"
		destPartnerLogin = "dest_partner_login"
		password         = "sesame"

		followID   = 123456
		msgContent = "transfer acknowledged"
	)

	// ############ SETUP LOCAL SERVER ###############
	server := &model.LocalAgent{Name: "server", Address: gwtesting.Address(t), Protocol: Pesit}
	require.NoError(t, db.Insert(server).Run())
	locAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: destPartnerLogin}
	require.NoError(t, db.Insert(locAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: locAccount.NullableID(),
		Type:           auth.Password, Value: password,
	}).Run())

	// ############ SETUP ORIGIN PARTNER ###############
	origPartnerServer := makeTestMessageServer(t)
	origPartner := &model.RemoteAgent{
		Name:        origPartnerName,
		Protocol:    Pesit,
		Address:     types.MustAddr(origPartnerServer.addr),
		ProtoConfig: map[string]any{"login": origPartnerLogin},
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
		Name:        destPartnerName,
		Protocol:    Pesit,
		Address:     types.Addr("localhost", 1234),
		ProtoConfig: map[string]any{"login": destPartnerLogin},
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
	transOrig := &model.HistoryEntry{
		ID:               1000,
		RemoteTransferID: "1000",
		Protocol:         Pesit,
		Status:           types.StatusDone,
		Client:           client.Name,
		Rule:             ruleRecv.Name,
		IsSend:           ruleRecv.IsSend,
		Account:          origAccount.Login,
		Agent:            origPartner.Name,
		SrcFilename:      "test.txt",
		Start:            time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local),
		Stop:             time.Date(2025, 1, 1, 1, 0, 0, 0, time.Local),
		TransferInfo:     map[string]any{model.FollowID: followID},
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
	conn, err := net.Dial("tcp", server.Address.String())
	require.NoError(t, err)
	require.NoError(t, pesitClient.Connect(conn))

	// ############ SEND MESSAGE ###############
	transferID, err := utils.ParseUint[uint32](transDest.RemoteTransferID)
	require.NoError(t, err)
	message, err := pesitClient.NewMessage(pesit.FileACK, transDest.SrcFilename, transferID)
	require.NoError(t, err)
	_, err = io.WriteString(message, msgContent)
	require.NoError(t, err)
	require.NoError(t, message.Close())

	// ############ CHECK ORIGIN ###############
	assert.Equal(t, origAccount.Login, origPartnerServer.login)
	assert.Equal(t, password, origPartnerServer.pswd)
	assert.Equal(t, pesit.FileACK, origPartnerServer.msgType)
	assert.Equal(t, transOrig.RemoteTransferID, utils.FormatUint(origPartnerServer.msgTransID))
	assert.Equal(t, transDest.SrcFilename, origPartnerServer.msgFilename)
	assert.Equal(t, msgContent, origPartnerServer.msgContent)

	// ############ CHECK TRANSFERS ###############
	var check model.NormalizedTransfers
	require.NoError(t, db.Select(&check).OrderBy("is_transfer", true).Eager().Run())
	require.Len(t, check, 2)

	assert.Equal(t, transOrig.ID, check[0].ID)
	assert.Subset(t, check[0].TransferInfo, map[string]any{
		ackSentKey: true,
	})

	assert.Equal(t, transDest.ID, check[1].ID)
	assert.Subset(t, check[1].TransferInfo, map[string]any{
		ackReceivedKey: true,
	})
}

func TestMessageHandlerServer(t *testing.T) {
	t.Parallel()

	// ############ SETUP CONFIG ###############
	db := dbtest.TestDatabase(t)

	const (
		origPartnerName  = "origin_partner"
		origPartnerLogin = "origin_partner_login"
		destPartnerName  = "dest_partner"
		destPartnerLogin = "dest_partner_login"
		password         = "sesame"

		followID   = 123456
		msgContent = "transfer acknowledged"
	)

	// ############ SETUP LOCAL SERVER ###############
	server := &model.LocalAgent{Name: "server", Address: gwtesting.Address(t), Protocol: Pesit}
	require.NoError(t, db.Insert(server).Run())
	destLocAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: destPartnerLogin}
	require.NoError(t, db.Insert(destLocAccount).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: destLocAccount.NullableID(),
		Type:           auth.Password,
		Value:          password,
	}).Run())
	origLocAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: origPartnerLogin}
	require.NoError(t, db.Insert(origLocAccount).Run())

	// ############ SETUP ORIGIN PARTNER ###############
	origPartnerServer := makeTestMessageServer(t)
	origPartner := &model.RemoteAgent{
		Name:        origPartnerName,
		Protocol:    Pesit,
		Address:     types.MustAddr(origPartnerServer.addr),
		ProtoConfig: map[string]any{"login": origPartnerLogin},
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
		Name:        destPartnerName,
		Protocol:    Pesit,
		Address:     types.Addr("localhost", 1234),
		ProtoConfig: map[string]any{"login": destPartnerLogin},
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
	conn, err := net.Dial("tcp", server.Address.String())
	require.NoError(t, err)
	require.NoError(t, pesitClient.Connect(conn))

	// ############ SEND MESSAGE ###############
	transferID, err := utils.ParseUint[uint32](transDest.RemoteTransferID)
	require.NoError(t, err)
	message, err := pesitClient.NewMessage(pesit.FileACK, transDest.SrcFilename, transferID)
	require.NoError(t, err)
	_, err = io.WriteString(message, msgContent)
	require.NoError(t, err)
	require.NoError(t, message.Close())

	// ############ CHECK ORIGIN ###############
	assert.Equal(t, origAccount.Login, origPartnerServer.login)
	assert.Equal(t, password, origPartnerServer.pswd)
	assert.Equal(t, pesit.FileACK, origPartnerServer.msgType)
	assert.Equal(t, transOrig.RemoteTransferID, utils.FormatUint(origPartnerServer.msgTransID))
	assert.Equal(t, transDest.SrcFilename, origPartnerServer.msgFilename)
	assert.Equal(t, msgContent, origPartnerServer.msgContent)

	// ############ CHECK TRANSFERS ###############
	var check model.NormalizedTransfers
	require.NoError(t, db.Select(&check).OrderBy("id", true).Eager().Run())
	require.Len(t, check, 2)

	assert.Equal(t, transOrig.ID, check[0].ID)
	assert.Subset(t, check[0].TransferInfo, map[string]any{
		ackSentKey: true,
	})

	assert.Equal(t, transDest.ID, check[1].ID)
	assert.Subset(t, check[1].TransferInfo, map[string]any{
		ackReceivedKey: true,
	})
}
