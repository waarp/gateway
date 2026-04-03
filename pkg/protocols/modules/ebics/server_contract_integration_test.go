package ebics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServerHTTPContractOrdersServeConfiguredViews(t *testing.T) {
	setEBICSConfigChecker(t)

	serverDB := dbtest.TestDatabase(t)
	serverRoot := t.TempDir()
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)

	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)

	serverHost := insertTestEbicsHost(t, serverDB, "HOST-CONTRACT")
	serverHost.DefaultBankURL = serverURL
	require.NoError(t, serverDB.Update(serverHost).Run())

	subscriber := insertTestEbicsSubscriber(t, serverDB, serverHost.ID, "PARTNER-CONTRACT", "USER-CONTRACT", true)
	subscriber.LocalAccountID = utils.NewNullInt64(serverAccount.ID)
	require.NoError(t, serverDB.Update(subscriber).Run())
	insertSubscriberKeyMaterial(
		t,
		serverDB,
		subscriber.ID,
		model.EbicsKeyUsageAuthenticationForRuntime(),
		"X002",
		testhelpers.ClientFooCert,
	)
	insertSubscriberKeyMaterial(
		t,
		serverDB,
		subscriber.ID,
		model.EbicsKeyUsageEncryptionForRuntime(),
		"E002",
		testhelpers.ClientFooCert2,
	)
	insertSubscriberKeyMaterial(
		t,
		serverDB,
		subscriber.ID,
		model.EbicsKeyUsageSignatureForRuntime(),
		"A006",
		testhelpers.ClientFooCert,
	)

	insertTestActiveServerContractSet(t, serverDB, serverHost.ID, sql.NullInt64{}, "HPD",
		&model.EbicsServerContractItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:RECOVERY",
			IsEnabled: false,
		},
		&model.EbicsServerContractItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:VERSIONS:PROTOCOL",
			Payload:   "H005",
			IsEnabled: true,
		},
	)
	insertTestActiveServerContractSet(t, serverDB, serverHost.ID, utils.NewNullInt64(subscriber.ID), "HKD",
		&model.EbicsServerContractItem{
			ItemType:      "BTF",
			ItemKey:       "HKD:BTD:MCT",
			OrderType:     "BTD",
			ServiceName:   "MCT",
			Scope:         "FR",
			MsgName:       "camt.054",
			ContainerType: "ZIP",
			IsEnabled:     true,
		},
	)
	insertTestActiveServerContractSet(t, serverDB, serverHost.ID, utils.NewNullInt64(subscriber.ID), "HTD",
		&model.EbicsServerContractItem{
			ItemType:           "PERMISSION",
			ItemKey:            "HTD:BTU:MCT",
			OrderType:          "BTU",
			ServiceName:        "MCT",
			Scope:              "FR",
			MsgName:            "pain.001",
			AuthorisationLevel: "A",
			AccountID:          "ACC-SERVER-1",
			MaxAmountValue:     "1000.00",
			MaxAmountCurrency:  "EUR",
			IsEnabled:          true,
		},
	)
	insertTestActiveServerContractSet(t, serverDB, serverHost.ID, sql.NullInt64{}, "HAA",
		&model.EbicsServerContractItem{
			ItemType:      "CAPABILITY",
			ItemKey:       "HAA:SERVICE:MCT",
			ServiceName:   "MCT",
			Scope:         "FR",
			MsgName:       "camt.054",
			ContainerType: "ZIP",
			IsEnabled:     true,
		},
	)

	clientDB := startSecondaryTestDatabase(t, "server_contract_orders_client")
	clientModel := &model.Client{
		Name:     "ebics-contract-client",
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())

	clientHost := insertTestEbicsHost(t, clientDB, "HOST-CONTRACT")
	clientHost.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(clientHost).Run())
	insertValidatedBankKey(t, clientDB, clientHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "contract-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "contract-trust", testhelpers.LocalhostCert)

	clientSubscriber := insertTestEbicsSubscriber(t, clientDB, clientHost.ID, "PARTNER-CONTRACT", "USER-CONTRACT", true)
	clientSubscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(clientSubscriber).Run())

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-contract",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-contract",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime(), authCred.ID)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageEncryptionForRuntime(), encCred.ID)

	clientService := NewClient(clientDB, clientModel)
	require.NoError(t, clientService.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = clientService.Stop(ctx)
	})

	execCtx, err := clientService.newAdminExecutionContext(clientSubscriber.ID)
	require.NoError(t, err)

	requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hpdDoc, _, err := execCtx.libClient.DownloadHPDDocument(
		requestCtx,
		libebicsclient.FlowHPDRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHPDOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, serverURL, hpdDoc.AccessParams.URLs[0].Value)
	assert.Equal(t, serverHost.HostID, hpdDoc.AccessParams.HostID)
	require.NotNil(t, hpdDoc.ProtocolParams.Recovery)
	assert.False(t, *hpdDoc.ProtocolParams.Recovery.Supported)

	hkdDoc, _, err := execCtx.libClient.DownloadHKDDocument(
		requestCtx,
		libebicsclient.FlowHKDRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHKDOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, hkdDoc.PartnerInfo.OrderInfo, 1)
	assert.Equal(t, "BTD", hkdDoc.PartnerInfo.OrderInfo[0].AdminOrderType)
	require.NotNil(t, hkdDoc.PartnerInfo.OrderInfo[0].Service)
	assert.Contains(t, hkdDoc.PartnerInfo.OrderInfo[0].Service.InnerXML, "<ServiceName>MCT</ServiceName>")
	require.Len(t, hkdDoc.UserInfo, 1)
	assert.Equal(t, "USER-CONTRACT", hkdDoc.UserInfo[0].UserID.Value)

	htdDoc, _, err := execCtx.libClient.DownloadHTDDocument(
		requestCtx,
		libebicsclient.FlowHTDRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHTDOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, htdDoc.PartnerInfo.AccountInfo, 1)
	assert.Equal(t, "ACC-SERVER-1", htdDoc.PartnerInfo.AccountInfo[0].ID)
	require.Len(t, htdDoc.UserInfo, 1)
	require.Len(t, htdDoc.UserInfo[0].Permission, 1)
	assert.Equal(t, "BTU", htdDoc.UserInfo[0].Permission[0].AdminOrderType)
	assert.Equal(t, "A", htdDoc.UserInfo[0].Permission[0].AuthorisationLevel)
	assert.Equal(t, "ACC-SERVER-1", htdDoc.UserInfo[0].Permission[0].AccountID)

	haaDoc, _, err := execCtx.libClient.DownloadHAADocument(
		requestCtx,
		libebicsclient.FlowHAARequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHAAOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, haaDoc.Service, 1)
	assert.Equal(t, "MCT", haaDoc.Service[0].ServiceName)
	assert.Equal(t, "camt.054", haaDoc.Service[0].MsgName.Value)
	require.NotNil(t, haaDoc.Service[0].Container)
	assert.Equal(t, "ZIP", haaDoc.Service[0].Container.ContainerType)
}

func insertTestActiveServerContractSet(
	t *testing.T,
	db *database.DB,
	hostPK int64,
	subscriberPK sql.NullInt64,
	sourceOrderType string,
	items ...*model.EbicsServerContractItem,
) *model.EbicsServerContractSet {
	t.Helper()

	set := &model.EbicsServerContractSet{
		EbicsHostID:       hostPK,
		EbicsSubscriberID: subscriberPK,
		SourceOrderType:   sourceOrderType,
		VersionTag:        "v1",
		Status:            "ACTIVE",
	}
	require.NoError(t, db.Insert(set).Run())

	for _, item := range items {
		item.ServerContractSetID = set.ID
		require.NoError(t, db.Insert(item).Run())
	}

	return set
}
