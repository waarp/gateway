package ebics

import (
	"context"
	"testing"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestControllerExecutesPlannedBTDToFinalPayload(t *testing.T) {
	setEBICSConfigChecker(t)

	clientRoot := t.TempDir()
	serverRoot := t.TempDir()

	oldPaths := conf.GlobalConfig.Paths
	oldController := conf.GlobalConfig.Controller
	conf.GlobalConfig.Paths.GatewayHome = clientRoot
	conf.GlobalConfig.Paths.DefaultInDir = "in"
	conf.GlobalConfig.Paths.DefaultOutDir = "out"
	conf.GlobalConfig.Paths.DefaultTmpDir = "tmp"
	conf.GlobalConfig.Controller.Delay = 20 * time.Millisecond
	t.Cleanup(func() {
		conf.GlobalConfig.Paths = oldPaths
		conf.GlobalConfig.Controller = oldController
	})

	serverDB := dbtest.TestDatabase(t)
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)
	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)
	prepareBankSideDownloadFixture(t, serverDB, serverRoot, serverAccount)

	clientDB := startSecondaryTestDatabase(t, "controller_planned_btd")
	clientName := "ebics-controller-planned-client"
	t.Cleanup(func() { delete(services.Clients, clientName) })

	host := insertTestEbicsHost(t, clientDB, "HOST-BANK")
	host.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(host).Run())
	insertValidatedBankKey(t, clientDB, host.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, host.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, host.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	rule := insertTestRule(t, clientDB, "controller-direct-download", false)
	rule.LocalDir = "incoming"
	rule.TmpLocalRcvDir = "tmp"
	require.NoError(t, clientDB.Update(rule).Run())

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "controller-direct-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "bank-trust-direct", testhelpers.LocalhostCert)

	subscriber := insertTestEbicsSubscriber(t, clientDB, host.ID, "PARTNER-BANK", "USER-BANK", true)
	subscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(subscriber).Run())

	insertTestPayloadProfile(t, clientDB, &model.EbicsPayloadProfile{
		Name:          "controller-direct-download-profile",
		OrderType:     "BTD",
		Direction:     "DOWNLOAD",
		ServiceName:   "MCT",
		MsgName:       "camt.054",
		DefaultRuleID: utils.NewNullInt64(rule.ID),
		IsEnabled:     true,
	})
	insertTestActiveContractItem(t, clientDB, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "ORDER_TYPE",
		ItemKey:     "btd-mct-controller-direct",
		OrderType:   "BTD",
		ServiceName: "MCT",
		MsgName:     "camt.054",
		IsEnabled:   true,
	})
	require.NoError(t, clientDB.Insert(&model.RuleAccess{
		RuleID:          rule.ID,
		RemoteAccountID: utils.NewNullInt64(remoteAccount.ID),
	}).Run())

	clientModel := &model.Client{
		Name:     clientName,
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())
	clientService := NewClient(clientDB, clientModel)
	require.NoError(t, clientService.Start())
	services.Clients[clientName] = clientService
	t.Cleanup(func() {
		delete(services.Clients, clientName)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = clientService.Stop(ctx)
	})

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-direct",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-direct",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime(), authCred.ID)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime(), encCred.ID)

	transfer := &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(clientModel.ID),
		RemoteAccountID: utils.NewNullInt64(remoteAccount.ID),
		SrcFilename:     "controller-planned-btd.xml",
		Start:           time.Now().UTC(),
		Status:          types.StatusPlanned,
		TransferInfo:    map[string]any{},
	}
	require.NoError(t, clientDB.Insert(transfer).Run())

	cont := &controller.Controller{DB: clientDB}
	require.NoError(t, cont.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = cont.Stop(ctx)
	})

	var operation model.EbicsOperation
	require.Eventually(t, func() bool {
		var operations model.EbicsOperations
		if err := clientDB.Select(&operations).
			Where("transfer_id=? OR metadata LIKE ?", transfer.ID, "%archivedTransferID%").
			Run(); err != nil {
			return false
		}
		if len(operations) != 1 {
			return false
		}

		operation = *operations[0]
		return operation.Status == model.EbicsOperationStatusCompletedForRuntime() &&
			readTransferInt64(operation.MetadataMap, operationMetadataKeyArchivedTransferID) > 0
	}, 8*time.Second, 50*time.Millisecond)

	historyID := requireArchivedTransferID(t, &operation)

	var history model.HistoryEntry
	require.NoError(t, clientDB.Get(&history, "id=?", historyID).Run())
	assert.Equal(t, types.StatusDone, history.Status)
	assert.Equal(t, "controller-planned-btd.xml", history.SrcFilename)
	assert.NotContains(t, history.TransferInfo, transferInfoKeyEbicsOperationID)
}

func TestServerHTTPDownloadThroughLibEBICSCreatesOperationAndHistory(t *testing.T) {
	setEBICSConfigChecker(t)

	serverDB := dbtest.TestDatabase(t)
	serverRoot := t.TempDir()
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)
	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)
	prepareBankSideDownloadFixture(t, serverDB, serverRoot, serverAccount)

	clientDB := startSecondaryTestDatabase(t, "server_http_download_client")
	clientModel := &model.Client{
		Name:     "ebics-http-download-client",
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())

	host := insertTestEbicsHost(t, clientDB, "HOST-BANK")
	host.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(host).Run())

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "http-download-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "http-download-trust", testhelpers.LocalhostCert)

	subscriber := insertTestEbicsSubscriber(t, clientDB, host.ID, "PARTNER-BANK", "USER-BANK", true)
	subscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(subscriber).Run())

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-http-download",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-http-download",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime(), authCred.ID)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime(), encCred.ID)

	clientService := NewClient(clientDB, clientModel)
	require.NoError(t, clientService.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = clientService.Stop(ctx)
	})

	execCtx, err := clientService.newAdminExecutionContext(subscriber.ID)
	require.NoError(t, err)

	payload, _, err := execCtx.libClient.DownloadBTD(
		context.Background(),
		libebicsclient.FlowBTDRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			OrderID:   "HTTP-DOWNLOAD-1",
			Params: &ebicsxml.BTDOrderParams{
				Service: ebicsxml.RestrictedService{
					ServiceName: "MCT",
					Scope:       "FR",
					MsgName:     ebicsxml.MessageType{Value: "camt.054"},
				},
			},
		},
		libebicsclient.FlowBTDOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []byte("<Document>rtn controller payload</Document>"), payload)

	operation := loadOperationByRequestID(t, serverDB, "HTTP-DOWNLOAD-1")
	require.Equal(t, model.EbicsOperationStatusCompletedForRuntime(), operation.Status)
	require.Equal(t, model.EbicsGatewayOutcomeSuccessForRuntime(), operation.GatewayOutcome)
	historyID := requireArchivedTransferID(t, operation)

	history := loadHistoryByID(t, serverDB, historyID)
	assert.Equal(t, types.StatusDone, history.Status)
	assert.Equal(t, "rtn-controller-download.xml", history.SrcFilename)
	assert.NotContains(t, history.TransferInfo, transferInfoKeyEbicsOperationID)
}
