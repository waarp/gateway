package tasks

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

func TestReplaceVarsAddsDedicatedEbicsVariables(t *testing.T) {
	db := dbtest.TestDatabase(t)

	client := &model.Client{Name: "ebics-client", Protocol: testProtocol}
	require.NoError(t, db.Insert(client).Run())

	partner := &model.RemoteAgent{
		Name:     "ebics-partner",
		Protocol: testProtocol,
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
		TransferInfo:    map[string]any{"foo": "bar"},
	}
	require.NoError(t, db.Insert(transfer).Run())

	host := &model.EbicsHost{
		Name:            "Host test",
		HostID:          "HOST-EBICS",
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
		DefaultBankURL:  "https://bank.example.test/ebics",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "Subscriber test",
		PartnerID:   "PARTNER-EBICS",
		UserID:      "USER-EBICS",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	event := &model.EbicsRTNEvent{
		Source:            "provider-ebics",
		EventID:           "evt-ebics",
		CorrelationID:     "corr-ebics",
		IdempotenceKey:    "provider-ebics:evt-ebics",
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
		TransactionID:     "TX-EBICS-001",
		RequestID:         "REQ-EBICS-001",
		CorrelationID:     "CORR-EBICS-001",
		EbicsVersion:      "H005",
		Status:            model.EbicsOperationStatusWaitingPayloadTransferForRuntime(),
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		GatewayOutcome:    model.EbicsGatewayOutcomePendingBankForRuntime(),
		RetryDecision:     model.EbicsRetryDecisionNoRetryForRuntime(),
		TransferID:        sql.NullInt64{Int64: transfer.ID, Valid: true},
		RTNEventID:        sql.NullInt64{Int64: event.ID, Valid: true},
		MetadataMap: map[string]any{
			"profileName":     "profile-ebics",
			"endpointURL":     "https://bank.override.test/ebics",
			"rtnProviderName": "provider-ebics",
			"rtnSource":       "source-ebics",
			"service": map[string]any{
				"serviceName": "MCT",
				"msgName":     "camt.054",
			},
		},
	}
	require.NoError(t, db.Insert(operation).Run())

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{
			ID:           transfer.ID,
			TransferInfo: map[string]any{"foo": "bar"},
		},
	}

	replaced, err := replaceVars(
		"#EBICS_OPERATION_ID#/#EBICS_ORDER_TYPE#/#EBICS_HOST_ID#/#EBICS_PROFILE_NAME#/#EBICS_ENDPOINT_URL#/#EBICS_RTN_PROVIDER_NAME#/#EBICS_SERVICE#",
		db,
		transCtx,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"1/BTD/HOST-EBICS/profile-ebics/https://bank.override.test/ebics/provider-ebics/map[msgName:camt.054 serviceName:MCT]",
		replaced,
	)
}
