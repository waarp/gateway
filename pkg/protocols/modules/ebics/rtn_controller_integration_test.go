package ebics

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestRTNControllerExecutesScheduledBTDToFinalPayload(t *testing.T) {
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

	clientDB := startSecondaryTestDatabase(t, "rtn_controller_client")
	clientName := "ebics-rtn-controller-client"
	t.Cleanup(func() { delete(services.Clients, clientName) })

	host := insertTestEbicsHost(t, clientDB, "HOST-BANK")
	host.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(host).Run())
	insertValidatedBankKey(t, clientDB, host.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, host.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, host.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	rule := insertTestRule(t, clientDB, "rtn-controller-download", false)
	rule.LocalDir = "incoming"
	rule.TmpLocalRcvDir = "tmp"
	require.NoError(t, clientDB.Update(rule).Run())

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "rtn-controller-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "bank-trust", testhelpers.LocalhostCert)

	subscriber := insertTestEbicsSubscriber(t, clientDB, host.ID, "PARTNER-BANK", "USER-BANK", true)
	subscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(subscriber).Run())

	insertTestPayloadProfile(t, clientDB, &model.EbicsPayloadProfile{
		Name:          "rtn-controller-download-profile",
		OrderType:     "BTD",
		Direction:     "DOWNLOAD",
		ServiceName:   "MCT",
		MsgName:       "camt.054",
		DefaultRuleID: utils.NewNullInt64(rule.ID),
		IsEnabled:     true,
	})
	insertTestActiveContractItem(t, clientDB, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "ORDER_TYPE",
		ItemKey:     "btd-mct-rtn-controller",
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

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime(), authCred.ID)
	insertActiveLifecycle(t, clientDB, subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime(), encCred.ID)

	provider := insertTestRTNProvider(t, clientDB, subscriber.ID, "AUTO", clientModel.ID)
	fake := newFakeRTNProvider()

	rtnService := NewRTNService(clientDB)
	rtnService.providerFactory = func(cfg *model.EbicsRTNProvider) (rtn.Provider, error) {
		require.Equal(t, provider.ID, cfg.ID)
		return fake, nil
	}
	require.NoError(t, rtnService.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = rtnService.Stop(ctx)
	})

	fake.events <- rtn.RawEvent{
		Source:        provider.Name,
		EventID:       "evt-controller-001",
		CorrelationID: "corr-controller-001",
		ReceivedAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"orderTypeHint": "BTD",
			"profileID":     "rtn-controller-download-profile",
			"msgName":       "camt.054",
			"outputName":    "rtn-controller-download.xml",
		},
	}

	var operation model.EbicsOperation
	require.Eventually(t, func() bool {
		err := clientDB.Get(&operation, "correlation_id=?", "corr-controller-001").Run()
		return err == nil
	}, 5*time.Second, 50*time.Millisecond)

	cont := &controller.Controller{DB: clientDB}
	require.NoError(t, cont.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = cont.Stop(ctx)
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		refreshed := model.EbicsOperation{}
		require.NoError(t, clientDB.Get(&refreshed, "id=?", operation.ID).Run())
		operation = refreshed
		if refreshed.Status == model.EbicsOperationStatusCompletedForRuntime() &&
			hasOperationHistoryLink(&refreshed) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	count, err := clientDB.Count(&model.EbicsOperation{}).Where("correlation_id=?", "corr-controller-001").Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, count, "unexpected duplicate EBICS operations for correlation")
	require.Equal(t, model.EbicsOperationStatusCompletedForRuntime(), operation.Status,
		"unexpected final EBICS operation state: outcome=%q technical=%q business=%q",
		operation.GatewayOutcome,
		operation.TechnicalReturnCode,
		operation.BusinessReturnCode,
	)
	require.True(t, hasOperationHistoryLink(&operation), "operation is completed but not yet linked to history")

	var history model.HistoryEntry
	require.NoError(t, clientDB.Get(&history, "id=?", requireOperationHistoryID(t, &operation)).Run())

	assert.Equal(t, types.StatusDone, history.Status)
	assert.Equal(t, "rtn-controller-download.xml", history.DestFilename)
	assert.NotContains(t, history.TransferInfo, transferInfoKeyEbicsOperationID)

	var event model.EbicsRTNEvent
	require.NoError(t, clientDB.Get(&event, "event_id=?", "evt-controller-001").Run())
	assert.Equal(t, "PROCESSED", event.Status)
	assert.False(t, event.ProcessedAt.IsZero())
	assert.Empty(t, event.LastError)
	assert.EqualValues(t, operation.ID, requireInt64Value(t, event.PayloadMap["autoPullOperationID"]))
	assert.EqualValues(t, history.ID, requireInt64Value(t, event.PayloadMap["autoPullTransferID"]))
	assert.Equal(t, model.EbicsOperationStatusCompletedForRuntime(), event.PayloadMap["autoPullStatus"])
	assert.Equal(t, model.EbicsGatewayOutcomeSuccessForRuntime(), event.PayloadMap["autoPullOutcome"])
	assert.Equal(t, model.EbicsRetryDecisionNoRetryForRuntime(), event.PayloadMap["autoPullRetry"])
}

func startSecondaryTestDatabase(t *testing.T, name string) *database.DB {
	t.Helper()

	conf.GlobalConfig.GatewayName = "gw-test"
	conf.GlobalConfig.Database = conf.DatabaseConfig{
		Type:    database.SQLite,
		Address: name,
	}
	conf.GlobalConfig.Paths.FilePerms = 0o600
	conf.GlobalConfig.Paths.DirPerms = 0o700

	db := &database.DB{}
	require.NoError(t, db.Start())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, db.Stop(ctx))
	})

	return db
}

func prepareBankSideDownloadFixture(
	t *testing.T,
	db *database.DB,
	rootDir string,
	account *model.LocalAccount,
) {
	t.Helper()

	sendRule := insertTestRule(t, db, "bank-download-rule", true)
	sendRule.LocalDir = "send"
	require.NoError(t, db.Update(sendRule).Run())

	host := insertTestEbicsHost(t, db, "HOST-BANK")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-BANK", "USER-BANK", true)
	subscriber.LocalAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())

	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "bank-download-profile",
		OrderType:     "BTD",
		Direction:     "DOWNLOAD",
		ServiceName:   "MCT",
		MsgName:       "camt.054",
		DefaultRuleID: utils.NewNullInt64(sendRule.ID),
		IsEnabled:     true,
	})
	insertTestActiveContractItem(t, db, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "ORDER_TYPE",
		ItemKey:     "btd-mct-bank",
		OrderType:   "BTD",
		ServiceName: "MCT",
		MsgName:     "camt.054",
		IsEnabled:   true,
	})
	insertSubscriberKeyMaterial(
		t,
		db,
		subscriber.ID,
		model.EbicsKeyUsageAuthenticationForRuntime(),
		"X002",
		testhelpers.ClientFooCert,
	)
	insertSubscriberKeyMaterial(
		t,
		db,
		subscriber.ID,
		model.EbicsKeyUsageEncryptionForRuntime(),
		"E002",
		testhelpers.ClientFooCert2,
	)
	insertSubscriberKeyMaterial(
		t,
		db,
		subscriber.ID,
		model.EbicsKeyUsageSignatureForRuntime(),
		"A006",
		testhelpers.ClientFooCert,
	)

	sourcePath := filepath.Join(rootDir, "send", "rtn-controller-download.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))
	payload := []byte("<Document>rtn controller payload</Document>")
	require.NoError(t, os.WriteFile(sourcePath, payload, 0o600))
	require.NoError(t, db.Insert(&model.Transfer{
		RuleID:         sendRule.ID,
		LocalAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:    "reserved-ebics-pipeline-id.xml",
		LocalPath:      filepath.Join(rootDir, "send", "reserved-ebics-pipeline-id.xml"),
		Start:          time.Now().UTC(),
		Status:         types.StatusPlanned,
		TransferInfo:   map[string]any{},
	}).Run())

	transfer := &model.Transfer{
		RuleID:         sendRule.ID,
		LocalAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:    "rtn-controller-download.xml",
		LocalPath:      sourcePath,
		Filesize:       int64(len(payload)),
		Start:          time.Now().UTC(),
		Status:         types.StatusAvailable,
		TransferInfo:   map[string]any{},
	}
	require.NoError(t, db.Insert(transfer).Run())
}

func insertTestRTNRemoteAccountForURL(
	t *testing.T,
	db *database.DB,
	login string,
	endpointURL string,
) *model.RemoteAccount {
	t.Helper()

	partner := &model.RemoteAgent{
		Name:     "partner-" + login,
		Protocol: EBICS,
		Address:  types.Addr("localhost", 443),
		ProtoConfig: map[string]any{
			"hostID":      "HOST-BANK",
			"endpointURL": endpointURL,
		},
	}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         login,
	}
	require.NoError(t, db.Insert(account).Run())

	return account
}

func requireOperationHistoryID(t *testing.T, operation *model.EbicsOperation) int64 {
	t.Helper()

	if operation.TransferID.Valid {
		return operation.TransferID.Int64
	}

	return requireArchivedTransferID(t, operation)
}

func hasOperationHistoryLink(operation *model.EbicsOperation) bool {
	if operation == nil {
		return false
	}

	if operation.TransferID.Valid {
		return true
	}

	_, ok := operation.MetadataMap[operationMetadataKeyArchivedTransferID]

	return ok
}

func insertTrustedTLSCredential(
	t *testing.T,
	db *database.DB,
	remoteAgentID int64,
	name, cert string,
) *model.Credential {
	t.Helper()

	cred := &model.Credential{
		RemoteAgentID: utils.NewNullInt64(remoteAgentID),
		Name:          name,
		Type:          auth.TLSTrustedCertificate,
		Value:         cert,
	}
	require.NoError(t, db.Insert(cred).Run())

	return cred
}

func insertTrustedTLSCredentialForLocalAccount(
	t *testing.T,
	db *database.DB,
	localAccountID int64,
	name, cert string,
) *model.Credential {
	t.Helper()

	cred := &model.Credential{
		LocalAccountID: utils.NewNullInt64(localAccountID),
		Name:           name,
		Type:           auth.TLSTrustedCertificate,
		Value:          cert,
	}
	require.NoError(t, db.Insert(cred).Run())

	return cred
}

func insertTLSCredentialForRemoteAccount(
	t *testing.T,
	db *database.DB,
	accountID int64,
	name, cert, key string,
) *model.Credential {
	t.Helper()

	cred := &model.Credential{
		RemoteAccountID: utils.NewNullInt64(accountID),
		Name:            name,
		Type:            auth.TLSCertificate,
		Value:           cert,
		Value2:          key,
	}
	require.NoError(t, db.Insert(cred).Run())

	return cred
}

func insertActiveLifecycle(
	t *testing.T,
	db *database.DB,
	subscriberID int64,
	usage string,
	credentialID int64,
) *model.EbicsKeyLifecycle {
	t.Helper()

	now := time.Now().UTC()
	lifecycle := &model.EbicsKeyLifecycle{
		EbicsSubscriberID:   subscriberID,
		KeyUsage:            usage,
		RotationType:        model.EbicsRotationTypeRotationForRuntime(),
		CoordinationID:      usage + "-coord",
		Status:              model.EbicsKeyLifecycleStatusActivatedForRuntime(),
		CurrentCredentialID: credentialID,
		RequestedAt:         now.Add(-2 * time.Minute),
		SentAt:              now.Add(-time.Minute),
		ActivatedAt:         now,
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	return lifecycle
}

func insertValidatedBankKey(
	t *testing.T,
	db *database.DB,
	hostID int64,
	keyType, version, publicKey string,
) *model.EbicsBankKey {
	t.Helper()

	sum := sha256.Sum256([]byte(publicKey))
	key := &model.EbicsBankKey{
		EbicsHostID:   hostID,
		KeyType:       keyType,
		Version:       version,
		PublicKey:     publicKey,
		PublicKeyHash: fmt.Sprintf("%x", sum[:]),
		State:         "validated",
		ValidFrom:     time.Now().UTC(),
	}
	require.NoError(t, db.Insert(key).Run())

	return key
}

func insertValidatedServerBankKey(
	t *testing.T,
	db *database.DB,
	hostID int64,
	keyType, version, certificatePEM string,
) *model.EbicsBankKey {
	t.Helper()

	chain, err := utils.ParsePEMCertChain(certificatePEM)
	require.NoError(t, err)
	require.NotEmpty(t, chain)

	var publicKey string
	switch keyType {
	case "AUTH":
		publicKey = fmt.Sprintf(
			`<AuthenticationPubKeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data><AuthenticationVersion>%s</AuthenticationVersion></AuthenticationPubKeyInfo>`,
			base64.StdEncoding.EncodeToString(chain[0].Raw),
			version,
		)
	case "ENCRYPT":
		publicKey = fmt.Sprintf(
			`<EncryptionPubKeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data><EncryptionVersion>%s</EncryptionVersion></EncryptionPubKeyInfo>`,
			base64.StdEncoding.EncodeToString(chain[0].Raw),
			version,
		)
	case "SIGNATURE":
		publicKey = fmt.Sprintf(
			`<ds:SignaturePubKeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data><SignatureVersion>%s</SignatureVersion></ds:SignaturePubKeyInfo>`,
			base64.StdEncoding.EncodeToString(chain[0].Raw),
			version,
		)
	default:
		t.Fatalf("unsupported server bank key type %q", keyType)
	}

	return insertValidatedBankKey(t, db, hostID, keyType, version, publicKey)
}

func insertSubscriberKeyMaterial(
	t *testing.T,
	db *database.DB,
	subscriberID int64,
	usage, version string,
	certificate string,
) *model.EbicsSubscriberKeyMaterial {
	t.Helper()

	material := &model.EbicsSubscriberKeyMaterial{
		EbicsSubscriberID:  subscriberID,
		KeyUsage:           usage,
		Certificate:        certificate,
		CertificateVersion: version,
		State:              "ACTIVE",
	}
	require.NoError(t, db.Insert(material).Run())

	return material
}
