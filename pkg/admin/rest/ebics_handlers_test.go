package rest

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestListEbicsPayloadsFiltersPayloadOrders(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-PAYLOAD", "PARTNER-PAYLOAD", "USER-PAYLOAD")

	payloadOp := insertRESTEbicsOperation(t, db, &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		OrderType:         "BTU",
		Direction:         "OUTBOUND",
		TransportMode:     "ASYNC",
		Status:            "PLANNED",
		Severity:          "INFO",
		GatewayOutcome:    "PENDING_BANK",
		RetryDecision:     "AUTO_RETRY_ALLOWED",
		CorrelationID:     "payload-op",
	})
	insertRESTEbicsOperation(t, db, &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypeReportingForRuntime(),
		OrderType:         "HEV",
		Direction:         "OUTBOUND",
		TransportMode:     "SYNC",
		Status:            "COMPLETED",
		Severity:          "INFO",
		GatewayOutcome:    "SUCCESS",
		RetryDecision:     "NO_RETRY",
		CorrelationID:     "reporting-op",
	})

	req := httptest.NewRequest(http.MethodGet, "/ebics/payloads", nil)
	w := httptest.NewRecorder()

	listEbicsPayloads(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string][]map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body["payloads"], 1)
	assert.EqualValues(t, payloadOp.ID, body["payloads"][0]["id"])
	assert.Equal(t, "BTU", body["payloads"][0]["orderType"])
}

func TestRecoverEbicsPayloadResetsOperationState(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-RECOVER", "PARTNER-RECOVER", "USER-RECOVER")

	operation := insertRESTEbicsOperation(t, db, &model.EbicsOperation{
		EbicsHostID:             host.ID,
		EbicsSubscriberID:       subscriber.ID,
		OperationType:           model.EbicsOperationTypePayloadForRuntime(),
		OrderType:               "BTD",
		Direction:               "INBOUND",
		TransportMode:           "ASYNC",
		Status:                  "FAILED",
		Severity:                "ERROR",
		GatewayOutcome:          "TECHNICAL_FATAL_FAILURE",
		RetryDecision:           "RECOVERY_REQUIRED",
		CorrelationID:           "recover-op",
		TechnicalReturnCode:     "091005",
		TechnicalReturnMessage:  "technical failure",
		BusinessReturnCode:      "090003",
		BusinessReturnMessage:   "business failure",
		ManualActionRequired:    true,
		StartedAt:               time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
		FinishedAt:              time.Date(2026, 3, 31, 9, 1, 0, 0, time.UTC),
		MetadataMap:             map[string]any{"existing": "value"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/ebics/payloads/1/recover",
		strings.NewReader(`{"reason":"resume transfer","metadata":{"origin":"operator"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_operation": "1"})
	w := httptest.NewRecorder()

	recoverEbicsPayload(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var refreshed model.EbicsOperation
	require.NoError(t, db.Get(&refreshed, "id=?", operation.ID).Run())
	assert.Equal(t, "READY", refreshed.Status)
	assert.Equal(t, "RECOVERY_REQUIRED", refreshed.RetryDecision)
	assert.Equal(t, "PENDING_BANK", refreshed.GatewayOutcome)
	assert.False(t, refreshed.ManualActionRequired)
	assert.Empty(t, refreshed.TechnicalReturnCode)
	assert.Empty(t, refreshed.BusinessReturnCode)
	assert.Zero(t, refreshed.StartedAt)
	assert.Zero(t, refreshed.FinishedAt)
	assert.Equal(t, "recover", refreshed.MetadataMap["lastPayloadAction"])
	assert.Equal(t, "resume transfer", refreshed.MetadataMap["lastPayloadActionReason"])
}

func TestGetEbicsOperationDetailUsesArchivedTransferLinkAndSegments(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-DETAIL", "PARTNER-DETAIL", "USER-DETAIL")

	operation := insertRESTEbicsOperation(t, db, &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		OrderType:         "BTU",
		Direction:         "OUTBOUND",
		TransportMode:     "ASYNC",
		Status:            "COMPLETED",
		Severity:          "INFO",
		GatewayOutcome:    "SUCCESS",
		RetryDecision:     "NO_RETRY",
		CorrelationID:     "detail-op",
		MetadataMap:       map[string]any{"archivedTransferID": 9876},
	})
	tx := insertRESTEbicsTransaction(t, db, &model.EbicsTransaction{
		EbicsOperationID:  sql.NullInt64{Int64: operation.ID, Valid: true},
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-DETAIL",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         "OUTBOUND",
		SegmentCount:      2,
		CurrentSegment:    2,
	})
	insertRESTEbicsSegment(t, db, &model.EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      2,
		SegmentStatus:      model.EbicsTransactionSegmentStatusCompletedForRuntime(),
		Checksum:           "seg-2",
	})
	insertRESTEbicsSegment(t, db, &model.EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      1,
		SegmentStatus:      model.EbicsTransactionSegmentStatusStoredForRuntime(),
		Checksum:           "seg-1",
	})

	req := httptest.NewRequest(http.MethodGet, "/ebics/operations/"+marshalID(operation.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_operation": marshalID(operation.ID)})
	w := httptest.NewRecorder()

	getEbicsOperation(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsOperationDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Links)
	require.NotNil(t, body.Links.TransferID)
	assert.EqualValues(t, 9876, *body.Links.TransferID)
	require.NotNil(t, body.Transaction)
	assert.Equal(t, "TX-DETAIL", body.Transaction.TransactionID)
	require.Len(t, body.Segments, 2)
	assert.Equal(t, 1, body.Segments[0].SegmentNumber)
	assert.Equal(t, 2, body.Segments[1].SegmentNumber)
}

func TestGetEbicsTransactionReturnsOrderedSegments(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-TX", "PARTNER-TX", "USER-TX")

	tx := insertRESTEbicsTransaction(t, db, &model.EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-ORDER",
		OrderType:         "BTD",
		Status:            "RECOVERING",
		Direction:         "INBOUND",
		SegmentCount:      3,
		CurrentSegment:    2,
	})
	insertRESTEbicsSegment(t, db, &model.EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      3,
		SegmentStatus:      model.EbicsTransactionSegmentStatusCompletedForRuntime(),
		Checksum:           "seg-3",
	})
	insertRESTEbicsSegment(t, db, &model.EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      1,
		SegmentStatus:      model.EbicsTransactionSegmentStatusStoredForRuntime(),
		Checksum:           "seg-1",
	})

	req := httptest.NewRequest(http.MethodGet, "/ebics/transactions/"+marshalID(tx.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_transaction": marshalID(tx.ID)})
	w := httptest.NewRecorder()

	getEbicsTransaction(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Transaction *api.OutEbicsTransaction          `json:"transaction"`
		Segments    []*api.OutEbicsTransactionSegment `json:"segments"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Transaction)
	assert.Equal(t, "TX-ORDER", body.Transaction.TransactionID)
	require.Len(t, body.Segments, 2)
	assert.Equal(t, 1, body.Segments[0].SegmentNumber)
	assert.Equal(t, 3, body.Segments[1].SegmentNumber)
}

func insertRESTEbicsHostAndSubscriber(
	t *testing.T,
	db *database.DB,
	hostID, partnerID, userID string,
) (*model.EbicsHost, *model.EbicsSubscriber) {
	t.Helper()

	host := &model.EbicsHost{
		Name:            hostID,
		HostID:          hostID,
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        partnerID + ":" + userID,
		PartnerID:   partnerID,
		UserID:      userID,
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	return host, subscriber
}

func insertRESTEbicsOperation(t *testing.T, db *database.DB, operation *model.EbicsOperation) *model.EbicsOperation {
	t.Helper()
	require.NoError(t, db.Insert(operation).Run())
	return operation
}

func insertRESTEbicsTransaction(t *testing.T, db *database.DB, tx *model.EbicsTransaction) *model.EbicsTransaction {
	t.Helper()
	require.NoError(t, db.Insert(tx).Run())
	return tx
}

func insertRESTEbicsSegment(
	t *testing.T,
	db *database.DB,
	segment *model.EbicsTransactionSegment,
) *model.EbicsTransactionSegment {
	t.Helper()
	require.NoError(t, db.Insert(segment).Run())
	return segment
}

func marshalID(id int64) string {
	return strconv.FormatInt(id, 10)
}
