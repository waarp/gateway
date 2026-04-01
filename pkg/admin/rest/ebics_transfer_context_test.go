package rest

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestDBTransferToRESTIncludesEbicsContext(t *testing.T) {
	db := dbtest.TestDatabase(t)

	client := &model.Client{Name: "ebics-client", Protocol: testProto1}
	require.NoError(t, db.Insert(client).Run())

	partner := &model.RemoteAgent{
		Name:     "ebics-partner",
		Protocol: testProto1,
		Address:  types.Addr("localhost", 443),
	}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "ebics-account",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{Name: "ebics-rule", IsSend: false, Path: "/ebics"}
	require.NoError(t, db.Insert(rule).Run())

	transfer := &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(client.ID),
		RemoteAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:     "payload.xml",
		DestFilename:    "payload.xml",
		Start:           time.Now().UTC(),
		TransferInfo:    map[string]any{},
	}
	require.NoError(t, db.Insert(transfer).Run())

	host := &model.EbicsHost{
		Name:            "bank-host",
		HostID:          "HOST-REST",
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
		DefaultBankURL:  "https://bank.rest.test/ebics",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "bank-subscriber",
		PartnerID:   "PARTNER-REST",
		UserID:      "USER-REST",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	event := &model.EbicsRTNEvent{
		Source:            "provider-rest",
		EventID:           "evt-rest",
		CorrelationID:     "corr-rest",
		IdempotenceKey:    "provider-rest:evt-rest",
		EbicsHostID:       sql.NullInt64{Int64: host.ID, Valid: true},
		EbicsSubscriberID: sql.NullInt64{Int64: subscriber.ID, Valid: true},
		Status:            "PROCESSING",
		Attempts:          1,
		ReceivedAt:        time.Now().UTC(),
		PayloadMap:        map[string]any{},
	}
	require.NoError(t, db.Insert(event).Run())

	operation := &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		OrderType:         "BTD",
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
		TransportMode:     model.EbicsTransportModeAutoTriggeredForRuntime(),
		TransactionID:     "TX-REST-001",
		RequestID:         "REQ-REST-001",
		CorrelationID:     "CORR-REST-001",
		EbicsVersion:      "H005",
		Status:            model.EbicsOperationStatusWaitingPayloadTransferForRuntime(),
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		GatewayOutcome:    model.EbicsGatewayOutcomePendingBankForRuntime(),
		RetryDecision:     model.EbicsRetryDecisionNoRetryForRuntime(),
		TransferID:        sql.NullInt64{Int64: transfer.ID, Valid: true},
		RTNEventID:        sql.NullInt64{Int64: event.ID, Valid: true},
		MetadataMap: map[string]any{
			"profileName":     "profile-rest",
			"endpointURL":     "https://bank.rest.override/ebics",
			"rtnProviderName": "provider-rest",
			"rtnSource":       "source-rest",
			"service": map[string]any{
				"serviceName": "MCT",
				"msgName":     "camt.054",
			},
		},
	}
	require.NoError(t, db.Insert(operation).Run())

	var view model.NormalizedTransferView
	require.NoError(t, db.Get(&view, "id=?", transfer.ID).Run())

	jsonTransfer := DBTransferToREST(db, &view)
	require.NotNil(t, jsonTransfer.EbicsContext)
	require.EqualValues(t, operation.ID, jsonTransfer.EbicsContext["operationID"])
	require.EqualValues(t, event.ID, jsonTransfer.EbicsContext["rtnEventID"])
	require.Equal(t, "BTD", jsonTransfer.EbicsContext["orderType"])
	require.Equal(t, "HOST-REST", jsonTransfer.EbicsContext["hostID"])
	require.Equal(t, "PARTNER-REST", jsonTransfer.EbicsContext["partnerID"])
	require.Equal(t, "USER-REST", jsonTransfer.EbicsContext["userID"])
	require.Equal(t, "REQ-REST-001", jsonTransfer.EbicsContext["requestID"])
	require.Equal(t, "CORR-REST-001", jsonTransfer.EbicsContext["correlationID"])
	require.Equal(t, "H005", jsonTransfer.EbicsContext["protocolVersion"])
	require.Equal(t, "profile-rest", jsonTransfer.EbicsContext["profileName"])
	require.Equal(t, "https://bank.rest.override/ebics", jsonTransfer.EbicsContext["endpointURL"])
	require.Equal(t, "provider-rest", jsonTransfer.EbicsContext["rtnProviderName"])
	require.Equal(t, "source-rest", jsonTransfer.EbicsContext["rtnSource"])
}
