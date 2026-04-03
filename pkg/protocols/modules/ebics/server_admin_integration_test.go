package ebics

import (
	"context"
	"testing"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServerHTTPKeyManagementOrdersUseOperationalSubscriberScope(t *testing.T) {
	setEBICSConfigChecker(t)

	serverDB := dbtest.TestDatabase(t)
	serverRoot := t.TempDir()
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)

	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)

	serverHost := insertTestEbicsHost(t, serverDB, "HOST-ADMIN")
	serverHost.DefaultBankURL = serverURL
	require.NoError(t, serverDB.Update(serverHost).Run())
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	serverSubscriber := insertTestEbicsSubscriber(t, serverDB, serverHost.ID, "PARTNER-ADMIN", "USER-ADMIN", true)
	serverSubscriber.LocalAccountID = utils.NewNullInt64(serverAccount.ID)
	require.NoError(t, serverDB.Update(serverSubscriber).Run())

	clientDB := startSecondaryTestDatabase(t, "server_admin_orders_client")
	clientModel := &model.Client{
		Name:     "ebics-admin-client",
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())

	clientHost := insertTestEbicsHost(t, clientDB, "HOST-ADMIN")
	clientHost.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(clientHost).Run())
	insertValidatedBankKey(t, clientDB, clientHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "admin-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "admin-trust", testhelpers.LocalhostCert)

	clientSubscriber := insertTestEbicsSubscriber(t, clientDB, clientHost.ID, "PARTNER-ADMIN", "USER-ADMIN", true)
	clientSubscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(clientSubscriber).Run())

	signatureCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-signature-admin",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-admin",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-admin",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageSignatureForRuntime(), signatureCred.ID)
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

	iniOrderData, err := clientService.buildInitializationOrderData(clientSubscriber, "INI")
	require.NoError(t, err)
	err = execCtx.libClient.UploadINI(
		requestCtx,
		libebicsclient.FlowINIRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowKeyMgmtOptional{},
		iniOrderData,
	)
	require.NoError(t, err)

	hiaOrderData, err := clientService.buildInitializationOrderData(clientSubscriber, "HIA")
	require.NoError(t, err)
	err = execCtx.libClient.UploadHIA(
		requestCtx,
		libebicsclient.FlowHIARequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowKeyMgmtOptional{},
		hiaOrderData,
	)
	require.NoError(t, err)

	hpbPayload, _, err := execCtx.libClient.DownloadHPB(
		requestCtx,
		libebicsclient.FlowHPBRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHPBOptional{
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)

	hpbDoc, err := ebicsxml.ParseHPBResponseOrderData(hpbPayload)
	require.NoError(t, err)
	assert.NotEmpty(t, hpbDoc.AuthenticationPubKeyInfo)
	assert.NotEmpty(t, hpbDoc.EncryptionPubKeyInfo)
	assert.NotEmpty(t, hpbDoc.SignaturePubKeyInfo)

	var materials model.EbicsSubscriberKeyMaterials
	require.NoError(t, serverDB.Select(&materials).
		Where("ebics_subscriber_id=? AND state=?", serverSubscriber.ID, "ACTIVE").
		OrderBy("id", true).
		Run())
	require.Len(t, materials, 3)
	assert.Contains(t, materials[0].PublicKey+materials[1].PublicKey+materials[2].PublicKey, "X509Certificate")
}

func TestServerHTTPKeyManagementRejectsDisabledSubscriber(t *testing.T) {
	setEBICSConfigChecker(t)

	serverDB := dbtest.TestDatabase(t)
	serverRoot := t.TempDir()
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)

	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)

	serverHost := insertTestEbicsHost(t, serverDB, "HOST-ADMIN-DISABLED")
	serverHost.DefaultBankURL = serverURL
	require.NoError(t, serverDB.Update(serverHost).Run())
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedServerBankKey(t, serverDB, serverHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	serverSubscriber := insertTestEbicsSubscriber(t, serverDB, serverHost.ID, "PARTNER-BLOCKED", "USER-BLOCKED", false)
	serverSubscriber.LocalAccountID = utils.NewNullInt64(serverAccount.ID)
	require.NoError(t, serverDB.Update(serverSubscriber).Run())

	clientDB := startSecondaryTestDatabase(t, "server_admin_disabled_client")
	clientModel := &model.Client{
		Name:     "ebics-admin-disabled-client",
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())

	clientHost := insertTestEbicsHost(t, clientDB, "HOST-ADMIN-DISABLED")
	clientHost.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(clientHost).Run())
	insertValidatedBankKey(t, clientDB, clientHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "admin-disabled-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "admin-disabled-trust", testhelpers.LocalhostCert)
	clientSubscriber := insertTestEbicsSubscriber(t, clientDB, clientHost.ID, "PARTNER-BLOCKED", "USER-BLOCKED", true)
	clientSubscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(clientSubscriber).Run())

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-disabled",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-disabled",
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

	_, _, err = execCtx.libClient.DownloadHPB(
		requestCtx,
		libebicsclient.FlowHPBRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHPBOptional{
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "061099")
}
