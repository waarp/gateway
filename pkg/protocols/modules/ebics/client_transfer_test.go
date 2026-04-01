package ebics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestTransferClientResolveTransactionIDOmitSyntheticValueForDownload(t *testing.T) {
	client := &transferClient{
		pip: &pipeline.Pipeline{
			TransCtx: &model.TransferContext{
				Transfer: &model.Transfer{
					RemoteTransferID: "corr-download-001",
					TransferInfo: map[string]any{
						transferInfoKeyEbicsOrderType: "BTD",
					},
				},
				Rule: &model.Rule{IsSend: false},
			},
		},
	}

	require.Empty(t, client.resolveTransactionID())
}

func TestTransferClientResolveTransactionIDKeepsPersistedValueForDownloadRecovery(t *testing.T) {
	client := &transferClient{
		pip: &pipeline.Pipeline{
			TransCtx: &model.TransferContext{
				Transfer: &model.Transfer{
					RemoteTransferID: "corr-download-002",
					TransferInfo: map[string]any{
						transferInfoKeyEbicsOrderType: "BTD",
						"ebicsTransactionID":          "TX-REAL-002",
					},
				},
				Rule: &model.Rule{IsSend: false},
			},
		},
	}

	require.Equal(t, "TX-REAL-002", client.resolveTransactionID())
}

func TestReadTransferInt64AcceptsJSONNumber(t *testing.T) {
	require.EqualValues(t, 42, readTransferInt64(map[string]any{
		transferInfoKeyEbicsOperationID: json.Number("42"),
	}, transferInfoKeyEbicsOperationID))
}

func TestTransferClientCreateOperationReusesScheduledOperation(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)

	host := insertTestEbicsHost(t, db, "HOST-CLIENT")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-CLIENT", "USER-CLIENT", true)
	rule := insertTestRule(t, db, "rtn-client-download", false)
	account := insertTestRTNAutoPullRemoteAccount(t, db, "rtn-client-account")
	subscriber.RemoteAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())
	clientAccount := insertTestRTNAutoPullClient(t, db, "ebics-rtn-client-test")

	transfer := &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(clientAccount.ID),
		RemoteAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:     "rtn-btd-request.xml",
		Start:           time.Now().UTC(),
		LocalPath:       t.TempDir(),
		TransferInfo: map[string]any{
			transferInfoKeyEbicsOrderType:  "BTD",
			transferInfoKeyEbicsRTNEventID: 7001,
		},
	}
	require.NoError(t, db.Insert(transfer).Run())

	event := &model.EbicsRTNEvent{
		Source:            "wss",
		EventID:           "evt-rtn-client-001",
		CorrelationID:     "corr-rtn-001",
		IdempotenceKey:    "rtn-client-001",
		EbicsHostID:       utils.NewNullInt64(host.ID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            "RECEIVED",
		ReceivedAt:        time.Now().UTC(),
	}
	require.NoError(t, db.Insert(event).Run())

	existing, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
		Owner:             conf.GlobalConfig.GatewayName,
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OrderType:         "BTD",
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
		TransportMode:     model.EbicsTransportModeAutoTriggeredForRuntime(),
		CorrelationID:     "corr-rtn-001",
		ResolvedRequest: &ebicsruntime.ResolvedPayloadRequest{
			OrderType:       "BTD",
			ResolvedService: ebicsruntime.PayloadServiceRef{OrderType: "BTD", ServiceName: "MCT"},
		},
	})
	require.NoError(t, err)
	existing.Status = model.EbicsOperationStatusWaitingPayloadTransferForRuntime()
	existing.RTNEventID = utils.NewNullInt64(event.ID)
	require.NoError(t, db.Insert(existing).Run())
	transfer.TransferInfo[transferInfoKeyEbicsOperationID] = existing.ID

	client := &transferClient{
		parent: &Client{
			db: db,
			config: &clientConfig{
				ProtocolVersion: "H005",
			},
		},
		pip: &pipeline.Pipeline{
			TransCtx: &model.TransferContext{
				Transfer: transfer,
				Rule:     rule,
			},
		},
	}

	resolved := &ebicsruntime.ResolvedPayloadRequest{
		OrderType:       "BTD",
		ResolvedService: ebicsruntime.PayloadServiceRef{OrderType: "BTD", ServiceName: "MCT"},
	}

	operation, err := client.createOperation(host, subscriber, resolved)
	require.NoError(t, err)
	require.Equal(t, existing.ID, operation.ID)
	require.Equal(t, "BTD", operation.OrderType)
	require.Equal(t, "RUNNING", operation.Status)
	require.Equal(t, "H005", operation.EbicsVersion)
	require.Equal(t, transfer.ID, operation.TransferID.Int64)
	require.True(t, operation.TransferID.Valid)
	require.NotZero(t, operation.StartedAt)
	require.EqualValues(t, event.ID, operation.RTNEventID.Int64)

	var refreshed model.EbicsOperation
	require.NoError(t, db.Get(&refreshed, "id=?", existing.ID).Run())
	require.Equal(t, transfer.ID, refreshed.TransferID.Int64)
	require.True(t, refreshed.TransferID.Valid)
	require.Equal(t, "RUNNING", refreshed.Status)

	count, err := db.Count(&model.EbicsOperation{}).Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func TestTransferClientSyncRTNEventExecutionStateMarksRetryableFailure(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	event := &model.EbicsRTNEvent{
		Source:         "wss",
		EventID:        "evt-rtn-sync-001",
		IdempotenceKey: "rtn-sync-001",
		Status:         "PROCESSING",
		Attempts:       1,
		ReceivedAt:     time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		PayloadMap:     map[string]any{},
	}
	require.NoError(t, db.Insert(event).Run())

	operation := &model.EbicsOperation{
		OrderType:      "BTD",
		Status:         model.EbicsOperationStatusFailedForRuntime(),
		GatewayOutcome: model.EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime(),
		RetryDecision:  model.EbicsRetryDecisionAutoRetryAllowedForRuntime(),
		RTNEventID:     utils.NewNullInt64(event.ID),
		TransferID:     utils.NewNullInt64(451),
	}
	operation.ID = 97
	operation.TechnicalReturnMessage = "temporary outage"

	client := &transferClient{
		parent: &Client{db: db},
		exec: &payloadExecution{
			operation: operation,
		},
	}

	require.NoError(t, client.syncRTNEventExecutionState())

	var refreshed model.EbicsRTNEvent
	require.NoError(t, db.Get(&refreshed, "id=?", event.ID).Run())
	assert.Equal(t, "RETRYABLE", refreshed.Status)
	assert.Equal(t, "temporary outage", refreshed.LastError)
	assert.False(t, refreshed.NextRetryAt.IsZero())
	assert.Zero(t, refreshed.ProcessedAt)
	assert.EqualValues(t, 97, requireInt64Value(t, refreshed.PayloadMap["autoPullOperationID"]))
	assert.EqualValues(t, 451, requireInt64Value(t, refreshed.PayloadMap["autoPullTransferID"]))
	assert.Equal(t, model.EbicsGatewayOutcomeTechnicalRetryableFailureForRuntime(), refreshed.PayloadMap["autoPullOutcome"])
	assert.Equal(t, model.EbicsRetryDecisionAutoRetryAllowedForRuntime(), refreshed.PayloadMap["autoPullRetry"])
}
