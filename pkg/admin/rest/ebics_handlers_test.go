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
	auth "code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
		EbicsHostID:            host.ID,
		EbicsSubscriberID:      subscriber.ID,
		OperationType:          model.EbicsOperationTypePayloadForRuntime(),
		OrderType:              "BTD",
		Direction:              "INBOUND",
		TransportMode:          "ASYNC",
		Status:                 "FAILED",
		Severity:               "ERROR",
		GatewayOutcome:         "TECHNICAL_FATAL_FAILURE",
		RetryDecision:          "RECOVERY_REQUIRED",
		CorrelationID:          "recover-op",
		TechnicalReturnCode:    "091005",
		TechnicalReturnMessage: "technical failure",
		BusinessReturnCode:     "090003",
		BusinessReturnMessage:  "business failure",
		ManualActionRequired:   true,
		StartedAt:              time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
		FinishedAt:             time.Date(2026, 3, 31, 9, 1, 0, 0, time.UTC),
		MetadataMap:            map[string]any{"existing": "value"},
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

	operation := insertRESTEbicsOperation(t, db, &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		OrderType:         "BTD",
		Direction:         "INBOUND",
		TransportMode:     "ASYNC",
		Status:            "RUNNING",
		Severity:          "INFO",
		GatewayOutcome:    "PENDING_BANK",
		RetryDecision:     "RECOVERY_REQUIRED",
		RequestID:         "REQ-TX",
		CorrelationID:     "CORR-TX",
	})

	tx := insertRESTEbicsTransaction(t, db, &model.EbicsTransaction{
		EbicsOperationID:  sql.NullInt64{Int64: operation.ID, Valid: true},
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

	var body api.OutEbicsTransactionDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Transaction)
	assert.Equal(t, "HOST-TX", body.HostID)
	assert.Equal(t, "PARTNER-TX", body.PartnerID)
	assert.Equal(t, "USER-TX", body.UserID)
	assert.Equal(t, "REQ-TX", body.RequestID)
	assert.Equal(t, "CORR-TX", body.CorrelationID)
	assert.Equal(t, "TX-ORDER", body.Transaction.TransactionID)
	require.Len(t, body.Segments, 2)
	assert.Equal(t, 1, body.Segments[0].SegmentNumber)
	assert.Equal(t, 3, body.Segments[1].SegmentNumber)
}

func TestGetEbicsContractViewReturnsItemsOrderedByID(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-CV", "PARTNER-CV", "USER-CV")

	view := &model.EbicsContractView{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: sql.NullInt64{Int64: subscriber.ID, Valid: true},
		SourceOrderType:   "HTD",
		VersionTag:        "v1",
		Status:            "ACTIVE",
		FetchedAt:         time.Date(2026, 3, 31, 15, 0, 0, 0, time.UTC),
	}
	require.NoError(t, db.Insert(view).Run())

	first := &model.EbicsContractViewItem{
		ContractViewID: view.ID,
		ItemType:       "ORDER_TYPE",
		ItemKey:        "B",
		OrderType:      "BTU",
		IsEnabled:      true,
	}
	second := &model.EbicsContractViewItem{
		ContractViewID: view.ID,
		ItemType:       "ORDER_TYPE",
		ItemKey:        "A",
		OrderType:      "BTD",
		IsEnabled:      true,
	}
	require.NoError(t, db.Insert(first).Run())
	require.NoError(t, db.Insert(second).Run())

	req := httptest.NewRequest(http.MethodGet, "/ebics/contract-views/"+marshalID(view.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"contract_view": marshalID(view.ID)})
	w := httptest.NewRecorder()

	getEbicsContractView(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		ContractView *api.OutEbicsContractView       `json:"contractView"`
		Items        []*api.OutEbicsContractViewItem `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.ContractView)
	assert.Equal(t, "HOST-CV", body.ContractView.HostID)
	assert.Equal(t, "PARTNER-CV", body.ContractView.PartnerID)
	require.Len(t, body.Items, 2)
	assert.EqualValues(t, first.ID, body.Items[0].ID)
	assert.EqualValues(t, second.ID, body.Items[1].ID)
}

func TestActOnEbicsKeyLifecycleUpdatesStatusAndEvidence(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-KL", "PARTNER-KL", "USER-KL")
	credential := insertRESTEbicsCredential(t, db)
	nextCredential := insertRESTEbicsCredential(t, db)

	lifecycle := &model.EbicsKeyLifecycle{
		EbicsSubscriberID:   subscriber.ID,
		KeyUsage:            "AUTHENTICATION",
		RotationType:        "ROTATION",
		Status:              "ORDER_PLANNED",
		CurrentCredentialID: credential.ID,
		NextCredentialID:    sql.NullInt64{Int64: nextCredential.ID, Valid: true},
		CoordinationID:      "coord-kl",
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	req := httptest.NewRequest(http.MethodPatch, "/ebics/key-lifecycles/"+marshalID(lifecycle.ID),
		strings.NewReader(`{"action":"MARK_SENT","operator":"ops","reason":"submitted","evidence":{"ticket":"INC-42"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_key_lifecycle": marshalID(lifecycle.ID)})
	w := httptest.NewRecorder()

	actOnEbicsKeyLifecycle(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var refreshed model.EbicsKeyLifecycle
	require.NoError(t, db.Get(&refreshed, "id=?", lifecycle.ID).Run())
	assert.Equal(t, "ORDER_SENT", refreshed.Status)
	assert.Equal(t, "ops", refreshed.Operator)
	assert.Equal(t, "submitted", refreshed.Reason)
	assert.False(t, refreshed.SentAt.IsZero())
	assert.Equal(t, "INC-42", refreshed.EvidenceMap["ticket"])

	reqGet := httptest.NewRequest(http.MethodGet, "/ebics/key-lifecycles/"+marshalID(lifecycle.ID), nil)
	reqGet = mux.SetURLVars(reqGet, map[string]string{"ebics_key_lifecycle": marshalID(lifecycle.ID)})
	wGet := httptest.NewRecorder()

	getEbicsKeyLifecycle(logger, db).ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)
	var body api.OutEbicsKeyLifecycle
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &body))
	assert.Equal(t, "ops", body.Operator)
	assert.Equal(t, "submitted", body.Reason)
	assert.Equal(t, "INC-42", body.Evidence["ticket"])
}

func TestActOnEbicsInitializationCancelsWorkflow(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-INIT", "PARTNER-INIT", "USER-INIT")

	workflow := &model.EbicsInitializationWorkflow{
		EbicsSubscriberID: subscriber.ID,
		Status:            "RUNNING",
		CurrentStep:       "INI_SENT",
	}
	require.NoError(t, db.Insert(workflow).Run())

	req := httptest.NewRequest(http.MethodPatch, "/ebics/initializations/"+marshalID(workflow.ID),
		strings.NewReader(`{"action":"CANCEL","operator":"ops","reason":"abort","evidence":{"ticket":"INIT-9"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_initialization": marshalID(workflow.ID)})
	w := httptest.NewRecorder()

	actOnEbicsInitialization(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var refreshed model.EbicsInitializationWorkflow
	require.NoError(t, db.Get(&refreshed, "id=?", workflow.ID).Run())
	assert.Equal(t, "CANCELLED", refreshed.Status)
	assert.Equal(t, "CANCELLED", refreshed.CurrentStep)
	assert.Equal(t, "ops", refreshed.Operator)
	assert.Equal(t, "abort", refreshed.Reason)
	assert.Equal(t, "INIT-9", refreshed.EvidenceMap["ticket"])

	reqGet := httptest.NewRequest(http.MethodGet, "/ebics/initializations/"+marshalID(workflow.ID), nil)
	reqGet = mux.SetURLVars(reqGet, map[string]string{"ebics_initialization": marshalID(workflow.ID)})
	wGet := httptest.NewRecorder()

	getEbicsInitialization(logger, db).ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)
	var body api.OutEbicsInitializationWorkflow
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &body))
	assert.Equal(t, "ops", body.Operator)
	assert.Equal(t, "abort", body.Reason)
	assert.Equal(t, "INIT-9", body.Evidence["ticket"])
}

func TestActOnEbicsRTNEventRetryUpdatesStatusAttemptsAndMetadata(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	event := insertRESTEbicsRTNEvent(t, db, &model.EbicsRTNEvent{
		Source:         "BANK_PUSH",
		IdempotenceKey: "rtn-retry",
		Status:         "RECEIVED",
		Attempts:       1,
		ReceivedAt:     time.Date(2026, 3, 31, 16, 0, 0, 0, time.UTC),
	})

	req := httptest.NewRequest(http.MethodPatch, "/ebics/rtn/events/"+marshalID(event.ID),
		strings.NewReader(`{"action":"RETRY","reason":"temporary outage","metadata":{"ticket":"RTN-42"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_event": marshalID(event.ID)})
	w := httptest.NewRecorder()

	actOnEbicsRTNEvent(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsRTNEvent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "RETRYABLE", body.Status)
	assert.Equal(t, 2, body.Attempts)
	require.NotNil(t, body.NextRetryAt)
	assert.Equal(t, "temporary outage", body.LastError)

	var refreshed model.EbicsRTNEvent
	require.NoError(t, db.Get(&refreshed, "id=?", event.ID).Run())
	assert.Equal(t, "RETRYABLE", refreshed.Status)
	assert.Equal(t, 2, refreshed.Attempts)
	assert.Equal(t, "RETRY", refreshed.PayloadMap["lastOperatorAction"])
	assert.Equal(t, "temporary outage", refreshed.PayloadMap["lastOperatorReason"])
	assert.Equal(t, "RTN-42", refreshed.PayloadMap["lastOperatorMetadata"].(map[string]any)["ticket"])

	reqGet := httptest.NewRequest(http.MethodGet, "/ebics/rtn/events/"+marshalID(event.ID), nil)
	reqGet = mux.SetURLVars(reqGet, map[string]string{"ebics_rtn_event": marshalID(event.ID)})
	wGet := httptest.NewRecorder()

	getEbicsRTNEvent(logger, db).ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)
	var detail api.OutEbicsRTNEvent
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &detail))
	assert.Equal(t, "RETRY", detail.OperatorAction)
	assert.Equal(t, "temporary outage", detail.OperatorReason)
	assert.Equal(t, "RTN-42", detail.OperatorMetadata["ticket"])
}

func TestActOnEbicsRTNEventRejectsUnsupportedAction(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	event := insertRESTEbicsRTNEvent(t, db, &model.EbicsRTNEvent{
		Source:         "BANK_PUSH",
		IdempotenceKey: "rtn-invalid",
		Status:         "RECEIVED",
		ReceivedAt:     time.Date(2026, 3, 31, 16, 5, 0, 0, time.UTC),
	})

	req := httptest.NewRequest(http.MethodPatch, "/ebics/rtn/events/"+marshalID(event.ID),
		strings.NewReader(`{"action":"DROP","reason":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_event": marshalID(event.ID)})
	w := httptest.NewRecorder()

	actOnEbicsRTNEvent(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"DROP" is not a supported EBICS RTN action`)

	var refreshed model.EbicsRTNEvent
	require.NoError(t, db.Get(&refreshed, "id=?", event.ID).Run())
	assert.Equal(t, "RECEIVED", refreshed.Status)
	assert.Zero(t, refreshed.Attempts)
}

func TestGetEbicsRTNEventExposesAutoPullLinksAndOutcome(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	event := insertRESTEbicsRTNEvent(t, db, &model.EbicsRTNEvent{
		Source:         "BANK_PUSH",
		IdempotenceKey: "rtn-autopull",
		Status:         "PROCESSED",
		Attempts:       1,
		ReceivedAt:     time.Date(2026, 4, 1, 16, 0, 0, 0, time.UTC),
		ProcessedAt:    time.Date(2026, 4, 1, 16, 1, 0, 0, time.UTC),
		PayloadMap: map[string]any{
			"autoPullOperationID": int64(910),
			"autoPullTransferID":  int64(911),
			"autoPullOrderType":   "BTD",
			"autoPullStatus":      "COMPLETED",
			"autoPullOutcome":     "SUCCESS",
			"autoPullRetry":       "NO_RETRY",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ebics/rtn/events/"+marshalID(event.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_event": marshalID(event.ID)})
	w := httptest.NewRecorder()

	getEbicsRTNEvent(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body api.OutEbicsRTNEvent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.AutoPullOperationID)
	require.NotNil(t, body.AutoPullTransferID)
	assert.EqualValues(t, 910, *body.AutoPullOperationID)
	assert.EqualValues(t, 911, *body.AutoPullTransferID)
	assert.Equal(t, "BTD", body.AutoPullOrderType)
	assert.Equal(t, "COMPLETED", body.AutoPullStatus)
	assert.Equal(t, "SUCCESS", body.AutoPullOutcome)
	assert.Equal(t, "NO_RETRY", body.AutoPullRetry)
}

func TestAddEbicsRTNProviderRequiresConfiguration(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-RTN-CONF", "PARTNER-RTN-CONF", "USER-RTN-CONF")

	req := httptest.NewRequest(http.MethodPost, "/ebics/rtn/providers",
		strings.NewReader(`{"name":"provider-a","transport":"WSS","subscriberID":`+marshalID(subscriber.ID)+`}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	addEbicsRTNProvider(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "the EBICS RTN provider configuration is missing")
}

func TestUpdateEbicsRTNProviderPreservesRuntimeFields(t *testing.T) {
	setRESTEBICSConfigChecker(t)

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-RTN-PROVIDER", "PARTNER-RTN-PROVIDER", "USER-RTN-PROVIDER")
	client := insertRESTEbicsClient(t, db, "ebics-rtn-provider-client")

	lastConnection := time.Date(2026, 3, 31, 16, 10, 0, 0, time.UTC)
	provider := insertRESTEbicsRTNProvider(t, db, &model.EbicsRTNProvider{
		Name:              "provider-runtime",
		Transport:         "WSS",
		Enabled:           true,
		EbicsSubscriberID: subscriber.ID,
		ConfigurationMap:  map[string]any{"endpoint": "wss://bank.example/rtn", "clientID": client.ID},
		AutoPullPolicy:    "MANUAL",
		LastConnectionAt:  lastConnection,
		LastError:         "previous timeout",
	})

	req := httptest.NewRequest(http.MethodPatch, "/ebics/rtn/providers/"+provider.Name,
		strings.NewReader(`{"autoPullPolicy":"AUTO","clientID":`+marshalID(client.ID)+`,"configuration":{"endpoint":"wss://bank.example/new"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_provider": provider.Name})
	w := httptest.NewRecorder()

	updateEbicsRTNProvider(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/ebics/rtn/providers/provider-runtime", w.Header().Get("Location"))

	var refreshed model.EbicsRTNProvider
	require.NoError(t, db.Get(&refreshed, "id=?", provider.ID).Run())
	assert.Equal(t, "AUTO", refreshed.AutoPullPolicy)
	assert.Equal(t, "wss://bank.example/new", refreshed.ConfigurationMap["endpoint"])
	assert.EqualValues(t, client.ID, refreshed.ConfigurationMap["clientID"])
	assert.Equal(t, lastConnection, refreshed.LastConnectionAt.UTC())
	assert.Equal(t, "previous timeout", refreshed.LastError)
}

func TestGetEbicsRTNProviderExposesClientSelectionAndActivationState(t *testing.T) {
	setRESTEBICSConfigChecker(t)

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-RTN-GET", "PARTNER-RTN-GET", "USER-RTN-GET")
	client := insertRESTEbicsClient(t, db, "ebics-rtn-get-client")

	provider := insertRESTEbicsRTNProvider(t, db, &model.EbicsRTNProvider{
		Name:              "provider-get",
		Transport:         "WSS",
		Enabled:           true,
		EbicsSubscriberID: subscriber.ID,
		ConfigurationMap:  map[string]any{"endpoint": "wss://bank.example/rtn", "clientID": client.ID},
		AutoPullPolicy:    "AUTO",
	})

	req := httptest.NewRequest(http.MethodGet, "/ebics/rtn/providers/"+provider.Name, nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_provider": provider.Name})
	w := httptest.NewRecorder()

	getEbicsRTNProvider(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsRTNProvider
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.ClientID)
	assert.EqualValues(t, client.ID, *body.ClientID)
	assert.Equal(t, client.Name, body.ClientName)
	assert.Equal(t, "READY_AUTO", body.ActivationStatus)
	assert.Empty(t, body.ActivationReason)
}

func TestGetEbicsRTNProviderExplainsBlockedActivationState(t *testing.T) {
	setRESTEBICSConfigChecker(t)

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-RTN-BLOCKED", "PARTNER-RTN-BLOCKED", "USER-RTN-BLOCKED")
	client := insertRESTEbicsClient(t, db, "ebics-rtn-blocked-client")

	provider := insertRESTEbicsRTNProvider(t, db, &model.EbicsRTNProvider{
		Name:              "provider-blocked",
		Transport:         "WSS",
		Enabled:           true,
		EbicsSubscriberID: subscriber.ID,
		ConfigurationMap:  map[string]any{"endpoint": "wss://bank.example/rtn", "clientID": client.ID},
		AutoPullPolicy:    "AUTO",
	})
	client.Disabled = true
	require.NoError(t, db.Update(client).Run())

	req := httptest.NewRequest(http.MethodGet, "/ebics/rtn/providers/"+provider.Name, nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_rtn_provider": provider.Name})
	w := httptest.NewRecorder()

	getEbicsRTNProvider(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsRTNProvider
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "BLOCKED", body.ActivationStatus)
	assert.Equal(t, "the RTN provider client "+marshalID(client.ID)+" is disabled", body.ActivationReason)
	assert.Equal(t, client.Name, body.ClientName)
}

func TestGetEbicsRuntimePolicyCreatesAndReturnsDefaultPolicy(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	req := httptest.NewRequest(http.MethodGet, "/ebics/runtime-policy", nil)
	w := httptest.NewRecorder()

	getEbicsRuntimePolicy(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsRuntimePolicy
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Enabled)
	assert.EqualValues(t, model.DefaultEbicsMaintenanceIntervalSeconds, body.MaintenanceIntervalSeconds)
	assert.EqualValues(t, model.DefaultEbicsTransactionRetentionSeconds, body.TransactionRetentionSeconds)
	assert.EqualValues(t, model.DefaultEbicsRTNEventRetentionSeconds, body.RTNEventRetentionSeconds)
}

func TestSetEbicsRuntimePolicyUpdatesSingleton(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	_, err := model.EnsureDefaultEbicsRuntimePolicy(db)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/ebics/runtime-policy",
		strings.NewReader(`{"enabled":false,"maintenanceIntervalSeconds":900,"transactionRetentionSeconds":1200,"rtnEventRetentionSeconds":1800}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setEbicsRuntimePolicy(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/ebics/runtime-policy", w.Header().Get("Location"))

	policy, err := model.EnsureDefaultEbicsRuntimePolicy(db)
	require.NoError(t, err)
	assert.False(t, policy.Enabled)
	assert.EqualValues(t, 900, policy.MaintenanceIntervalSeconds)
	assert.EqualValues(t, 1200, policy.TransactionRetentionSeconds)
	assert.EqualValues(t, 1800, policy.RTNEventRetentionSeconds)
}

func TestAddEbicsContractRefreshPolicyCreatesPolicy(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	client := insertRESTEbicsClient(t, db, "ebics-contract-refresh-client")
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-CRP", "PARTNER-CRP", "USER-CRP")
	remoteAgent := &model.RemoteAgent{Name: "remote-agent-crp", Protocol: "https", Address: types.Addr("127.0.0.1", 443)}
	require.NoError(t, db.Insert(remoteAgent).Run())
	remoteAccount := &model.RemoteAccount{Login: "remote-account-crp", RemoteAgentID: remoteAgent.ID}
	require.NoError(t, db.Insert(remoteAccount).Run())
	subscriber.RemoteAccountID = sql.NullInt64{Int64: remoteAccount.ID, Valid: true}
	require.NoError(t, db.Update(subscriber).Run())

	req := httptest.NewRequest(http.MethodPost, "/ebics/contract-refresh-policies",
		strings.NewReader(`{"name":"daily-bank-a","clientID":`+marshalID(client.ID)+`,"subscriberID":`+marshalID(subscriber.ID)+`,"includeHEV":false,"intervalSeconds":3600}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	addEbicsContractRefreshPolicy(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/ebics/contract-refresh-policies/daily-bank-a", w.Header().Get("Location"))

	var policy model.EbicsContractRefreshPolicy
	require.NoError(t, db.Get(&policy, "name=?", "daily-bank-a").Run())
	assert.Equal(t, client.ID, policy.ClientID)
	assert.Equal(t, subscriber.ID, policy.EbicsSubscriberID)
	assert.EqualValues(t, 3600, policy.IntervalSeconds)
	assert.False(t, policy.IncludeHEV)
	assert.Equal(t, host.ID, subscriber.EbicsHostID)
}

func TestGetEbicsContractRefreshPolicyDisplaysActivationState(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	client := insertRESTEbicsClient(t, db, "ebics-contract-refresh-client-get")
	host, subscriber := insertRESTEbicsHostAndSubscriber(t, db, "HOST-CRP-GET", "PARTNER-CRP-GET", "USER-CRP-GET")
	remoteAgent := &model.RemoteAgent{Name: "remote-agent-crp-get", Protocol: "https", Address: types.Addr("127.0.0.1", 443)}
	require.NoError(t, db.Insert(remoteAgent).Run())
	remoteAccount := &model.RemoteAccount{Login: "remote-account-crp-get", RemoteAgentID: remoteAgent.ID}
	require.NoError(t, db.Insert(remoteAccount).Run())
	subscriber.RemoteAccountID = sql.NullInt64{Int64: remoteAccount.ID, Valid: true}
	require.NoError(t, db.Update(subscriber).Run())
	require.NoError(t, db.Insert(&model.EbicsContractRefreshPolicy{
		Name:              "daily-bank-b",
		Enabled:           true,
		ClientID:          client.ID,
		EbicsSubscriberID: subscriber.ID,
		IncludeHEV:        true,
		IntervalSeconds:   7200,
	}).Run())

	req := httptest.NewRequest(http.MethodGet, "/ebics/contract-refresh-policies/daily-bank-b", nil)
	req = mux.SetURLVars(req, map[string]string{"ebics_contract_refresh_policy": "daily-bank-b"})
	w := httptest.NewRecorder()

	getEbicsContractRefreshPolicy(logger, db).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body api.OutEbicsContractRefreshPolicy
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, client.ID, body.ClientID)
	assert.Equal(t, client.Name, body.ClientName)
	assert.Equal(t, host.HostID, body.HostID)
	assert.Equal(t, "READY", body.ActivationStatus)
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

func insertRESTEbicsCredential(t *testing.T, db *database.DB) *model.Credential {
	t.Helper()

	now := time.Now().UTC().UnixNano()
	suffix := strconv.FormatInt(now, 10)
	port := uint16(10000 + (now % 50000))
	agent := &model.LocalAgent{
		Name:     "ebics-rest-cred-server-" + suffix,
		Protocol: testProto1,
		Address:  types.Addr("localhost", port),
	}
	require.NoError(t, db.Insert(agent).Run())

	credential := &model.Credential{
		LocalAgentID: utils.NewNullInt64(agent.ID),
		Name:         "ebics-rest-credential-" + suffix,
		Type:         auth.Password,
		Value:        "sesame",
	}
	require.NoError(t, db.Insert(credential).Run())

	return credential
}

func insertRESTEbicsRTNEvent(t *testing.T, db *database.DB, event *model.EbicsRTNEvent) *model.EbicsRTNEvent {
	t.Helper()
	require.NoError(t, db.Insert(event).Run())
	return event
}

func insertRESTEbicsRTNProvider(
	t *testing.T,
	db *database.DB,
	provider *model.EbicsRTNProvider,
) *model.EbicsRTNProvider {
	t.Helper()
	require.NoError(t, db.Insert(provider).Run())
	return provider
}

func setRESTEBICSConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = nil
	modeltest.AddDummyProtoConfig("ebics")
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

func insertRESTEbicsClient(t *testing.T, db *database.DB, name string) *model.Client {
	t.Helper()

	client := &model.Client{
		Name:     name,
		Protocol: "ebics",
		ProtoConfig: map[string]any{
			"endpointURL": "https://bank.example.test/ebics",
		},
	}
	require.NoError(t, db.Insert(client).Run())

	return client
}

func marshalID(id int64) string {
	return strconv.FormatInt(id, 10)
}
