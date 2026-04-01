package ebics

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestServerIntegrationUploadCreatesOperationAndHistory(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	rootDir := t.TempDir()
	service, account := startTestEBICSIntegrationServer(t, db, rootDir)
	router := newPayloadOrderRouter(db, service.logger)

	receiveRule := insertTestRule(t, db, "ebics-upload-rule", false)
	receiveRule.LocalDir = "incoming"
	require.NoError(t, db.Update(receiveRule).Run())

	host := insertTestEbicsHost(t, db, "HOST-UPLOAD")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-UPLOAD", "USER-UPLOAD", true)
	subscriber.LocalAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())

	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "upload-profile",
		OrderType:     "BTU",
		Direction:     "UPLOAD",
		ServiceName:   "MCT",
		DefaultRuleID: utils.NewNullInt64(receiveRule.ID),
		IsEnabled:     true,
	})
	insertTestActiveContractItem(t, db, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "BTF",
		ItemKey:     "btu-mct-upload",
		OrderType:   "BTU",
		ServiceName: "MCT",
		IsEnabled:   true,
	})

	payload := []byte("<Document>upload payload</Document>")
	req := libebics.OrderContext{
		HostID:          "HOST-UPLOAD",
		PartnerID:       "PARTNER-UPLOAD",
		UserID:          "USER-UPLOAD",
		OrderType:       "BTU",
		OrderID:         "ORDER-UPLOAD-1",
		ProtocolVersion: "H005",
		TransID:         "TX-UPLOAD-1",
		PayloadRaw:      payload,
	}

	err := router.Upload(context.Background(), req, &ebicsxml.BTUOrderParams{
		FileName: "payload-upload.xml",
		Service: ebicsxml.RestrictedService{
			ServiceName: "MCT",
			Scope:       "FR",
			MsgName: ebicsxml.MessageType{
				Value: "pain.001",
			},
		},
	})
	require.NoError(t, err)

	operation := loadOperationByRequestID(t, db, "ORDER-UPLOAD-1")
	require.Equal(t, model.EbicsOperationStatusCompletedForRuntime(), operation.Status)
	require.Equal(t, model.EbicsGatewayOutcomeSuccessForRuntime(), operation.GatewayOutcome)
	require.Equal(t, model.EbicsOperationDirectionInboundForRuntime(), operation.Direction)
	require.False(t, operation.TransferID.Valid)
	require.NotZero(t, operation.FinishedAt)

	archivedTransferID := requireArchivedTransferID(t, operation)
	require.Zero(t, countTransfersByID(t, db, archivedTransferID))

	history := loadHistoryByID(t, db, archivedTransferID)
	require.Equal(t, types.StatusDone, history.Status)
	require.Equal(t, "payload-upload.xml", history.DestFilename)
	require.Equal(t, account.Login, history.Account)
	require.Equal(t, service.server.Name, history.Agent)
	require.Equal(t, EBICS, history.Protocol)
	require.NotContains(t, history.TransferInfo, transferInfoKeyEbicsOperationID)

	storedPayload, readErr := os.ReadFile(history.LocalPath)
	require.NoError(t, readErr)
	require.Equal(t, payload, storedPayload)
}

func TestServerIntegrationDownloadReturnsPayloadAndArchivesTransfer(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	rootDir := t.TempDir()
	service, account := startTestEBICSIntegrationServer(t, db, rootDir)
	router := newPayloadOrderRouter(db, service.logger)

	sendRule := insertTestRule(t, db, "ebics-download-rule", true)
	sendRule.LocalDir = "outgoing"
	require.NoError(t, db.Update(sendRule).Run())

	host := insertTestEbicsHost(t, db, "HOST-DOWNLOAD")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-DOWNLOAD", "USER-DOWNLOAD", true)
	subscriber.LocalAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())

	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "download-profile",
		OrderType:     "BTD",
		Direction:     "DOWNLOAD",
		ServiceName:   "MCT",
		DefaultRuleID: utils.NewNullInt64(sendRule.ID),
		IsEnabled:     true,
	})
	insertTestActiveContractItem(t, db, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "BTF",
		ItemKey:     "btd-mct-download",
		OrderType:   "BTD",
		ServiceName: "MCT",
		IsEnabled:   true,
	})

	sourcePath := filepath.Join(rootDir, "outgoing", "payload-download.xml")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))

	expectedPayload := []byte("<Document>download payload</Document>")
	require.NoError(t, os.WriteFile(sourcePath, expectedPayload, 0o600))

	transfer := &model.Transfer{
		RuleID:         sendRule.ID,
		LocalAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:    "payload-download.xml",
		LocalPath:      sourcePath,
		Filesize:       int64(len(expectedPayload)),
		Start:          time.Now().UTC(),
		Status:         types.StatusAvailable,
		TransferInfo:   map[string]any{},
	}
	require.NoError(t, db.Insert(transfer).Run())

	result, err := router.Download(context.Background(), libebics.OrderContext{
		HostID:          "HOST-DOWNLOAD",
		PartnerID:       "PARTNER-DOWNLOAD",
		UserID:          "USER-DOWNLOAD",
		OrderType:       "BTD",
		OrderID:         "ORDER-DOWNLOAD-1",
		ProtocolVersion: "H005",
		TransID:         "TX-DOWNLOAD-1",
	}, &ebicsxml.BTDOrderParams{
		Service: ebicsxml.RestrictedService{
			ServiceName: "MCT",
			Scope:       "FR",
			MsgName: ebicsxml.MessageType{
				Value: "camt.054",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, expectedPayload, result.PayloadRaw)

	operation := loadOperationByRequestID(t, db, "ORDER-DOWNLOAD-1")
	require.Equal(t, model.EbicsOperationStatusCompletedForRuntime(), operation.Status)
	require.Equal(t, model.EbicsGatewayOutcomeSuccessForRuntime(), operation.GatewayOutcome)
	require.Equal(t, model.EbicsOperationDirectionOutboundForRuntime(), operation.Direction)
	require.False(t, operation.TransferID.Valid)
	require.Equal(t, transfer.ID, requireArchivedTransferID(t, operation))

	require.Zero(t, countTransfersByID(t, db, transfer.ID))

	history := loadHistoryByID(t, db, transfer.ID)
	require.Equal(t, types.StatusDone, history.Status)
	require.Equal(t, "payload-download.xml", history.SrcFilename)
	require.Equal(t, account.Login, history.Account)
	require.NotContains(t, history.TransferInfo, transferInfoKeyEbicsOperationID)
}

func startTestEBICSIntegrationServer(t *testing.T, db *database.DB, rootDir string) (*Server, *model.LocalAccount) {
	t.Helper()

	agent := insertTestEBICSServer(t, db, gwtesting.GetLocalPort(t))
	agent.RootDir = rootDir
	agent.ReceiveDir = "receive"
	agent.SendDir = "send"
	agent.TmpReceiveDir = "tmp"
	require.NoError(t, db.Update(agent).Run())

	account := insertTestLocalAccount(t, db, agent.ID, "ebics-integration")
	service := NewServer(db, agent)
	require.NoError(t, service.Start())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, service.Stop(ctx))
	})

	return service, account
}

func loadOperationByRequestID(t *testing.T, db *database.DB, requestID string) *model.EbicsOperation {
	t.Helper()

	operation := &model.EbicsOperation{}
	require.NoError(t, db.Get(operation, "request_id=?", requestID).Run())

	return operation
}

func loadHistoryByID(t *testing.T, db *database.DB, id int64) *model.HistoryEntry {
	t.Helper()

	history := &model.HistoryEntry{}
	require.NoError(t, db.Get(history, "id=?", id).Run())

	return history
}

func countTransfersByID(t *testing.T, db *database.DB, id int64) uint64 {
	t.Helper()

	count, err := db.Count(&model.Transfer{}).Where("id=?", id).Run()
	require.NoError(t, err)

	return count
}

func requireArchivedTransferID(t *testing.T, operation *model.EbicsOperation) int64 {
	t.Helper()

	raw, ok := operation.MetadataMap[operationMetadataKeyArchivedTransferID]
	require.True(t, ok)

	switch value := raw.(type) {
	case int64:
		return value
	case float64:
		return int64(value)
	case json.Number:
		parsed, err := value.Int64()
		require.NoError(t, err)
		return parsed
	default:
		t.Fatalf("unexpected archived transfer ID type %T", raw)
		return 0
	}
}

func requireInt64Value(t *testing.T, raw any) int64 {
	t.Helper()

	switch value := raw.(type) {
	case int64:
		return value
	case float64:
		return int64(value)
	case json.Number:
		parsed, err := value.Int64()
		require.NoError(t, err)
		return parsed
	default:
		t.Fatalf("unexpected int64-like value type %T", raw)
		return 0
	}
}
