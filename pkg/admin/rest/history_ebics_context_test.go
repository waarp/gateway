package rest

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestFromHistoryWithDBIncludesEbicsContext(t *testing.T) {
	db := dbtest.TestDatabase(t)

	host := &model.EbicsHost{
		Name:            "bank-host-history",
		HostID:          "HOST-HISTORY",
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
		DefaultBankURL:  "https://bank.history.test/ebics",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "bank-subscriber-history",
		PartnerID:   "PARTNER-HISTORY",
		UserID:      "USER-HISTORY",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	event := &model.EbicsRTNEvent{
		Source:            "provider-history",
		EventID:           "evt-history",
		CorrelationID:     "corr-history-event",
		IdempotenceKey:    "provider-history:evt-history",
		EbicsHostID:       sql.NullInt64{Int64: host.ID, Valid: true},
		EbicsSubscriberID: sql.NullInt64{Int64: subscriber.ID, Valid: true},
		Status:            "PROCESSED",
		Attempts:          1,
		ReceivedAt:        time.Now().UTC(),
		ProcessedAt:       time.Now().UTC(),
		PayloadMap:        map[string]any{},
	}
	require.NoError(t, db.Insert(event).Run())

	history := &model.HistoryEntry{
		ID:               42,
		RemoteTransferID: "4200",
		IsServer:         false,
		IsSend:           false,
		Rule:             "ebics-history-rule",
		Account:          "ebics-history-account",
		Agent:            "ebics-history-agent",
		Client:           "ebics-history-client",
		Protocol:         "ebics",
		SrcFilename:      "payload.zip",
		DestFilename:     "payload.zip",
		LocalPath:        localPath("/history/payload.zip"),
		RemotePath:       "/remote/payload.zip",
		Filesize:         512,
		Start:            time.Now().UTC().Add(-time.Minute),
		Stop:             time.Now().UTC(),
		Status:           "DONE",
		TransferInfo:     map[string]any{"businessKey": "value"},
	}
	require.NoError(t, db.Insert(history).Run())

	operation := &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		OrderType:         "BTD",
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
		TransportMode:     model.EbicsTransportModeAutoTriggeredForRuntime(),
		TransactionID:     "TX-HISTORY-001",
		RequestID:         "REQ-HISTORY-001",
		CorrelationID:     "CORR-HISTORY-001",
		EbicsVersion:      "H005",
		Status:            model.EbicsOperationStatusCompletedForRuntime(),
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		GatewayOutcome:    model.EbicsGatewayOutcomeSuccessForRuntime(),
		RetryDecision:     model.EbicsRetryDecisionNoRetryForRuntime(),
		RTNEventID:        sql.NullInt64{Int64: event.ID, Valid: true},
		MetadataMap: map[string]any{
			"archivedTransferID": int64(history.ID),
			"profileName":     "profile-history",
			"endpointURL":     "https://bank.history.override/ebics",
			"rtnProviderName": "provider-history",
			"rtnSource":       "source-history",
			"service": map[string]any{
				"serviceName": "MCT",
				"msgName":     "camt.054",
			},
		},
	}
	require.NoError(t, db.Insert(operation).Run())

	out := FromHistoryWithDB(db, history)
	require.NotNil(t, out.EbicsContext)
	require.EqualValues(t, operation.ID, out.EbicsContext["operationID"])
	require.EqualValues(t, event.ID, out.EbicsContext["rtnEventID"])
	require.Equal(t, "BTD", out.EbicsContext["orderType"])
	require.Equal(t, "HOST-HISTORY", out.EbicsContext["hostID"])
	require.Equal(t, "PARTNER-HISTORY", out.EbicsContext["partnerID"])
	require.Equal(t, "USER-HISTORY", out.EbicsContext["userID"])
	require.Equal(t, "REQ-HISTORY-001", out.EbicsContext["requestID"])
	require.Equal(t, "CORR-HISTORY-001", out.EbicsContext["correlationID"])
	require.Equal(t, "H005", out.EbicsContext["protocolVersion"])
	require.Equal(t, "profile-history", out.EbicsContext["profileName"])
	require.Equal(t, "https://bank.history.override/ebics", out.EbicsContext["endpointURL"])
	require.Equal(t, "provider-history", out.EbicsContext["rtnProviderName"])
	require.Equal(t, "source-history", out.EbicsContext["rtnSource"])
	require.Equal(t, map[string]any{
		"serviceName": "MCT",
		"msgName":     "camt.054",
	}, out.EbicsContext["service"])
	require.Equal(t, map[string]any{"businessKey": "value"}, out.TransferInfo)
}
