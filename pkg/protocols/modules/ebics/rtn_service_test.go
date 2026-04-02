package ebics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type fakeRTNProvider struct {
	events  chan rtn.RawEvent
	errors  chan error
	started bool
}

func newFakeRTNProvider() *fakeRTNProvider {
	return &fakeRTNProvider{
		events: make(chan rtn.RawEvent, 8),
		errors: make(chan error, 8),
	}
}

func (p *fakeRTNProvider) Start(context.Context) error {
	p.started = true
	return nil
}

func (p *fakeRTNProvider) Stop(context.Context) error {
	p.started = false
	return nil
}

func (*fakeRTNProvider) State() (string, string) {
	return rtn.RTNProviderStateRunning, ""
}

func (p *fakeRTNProvider) Events() <-chan rtn.RawEvent {
	return p.events
}

func (p *fakeRTNProvider) Errors() <-chan error {
	return p.errors
}

func TestRTNServiceStartStopWithoutProviders(t *testing.T) {
	db := dbtest.TestDatabase(t)
	service := NewRTNService(db)

	require.NoError(t, service.Start())
	code, reason := service.State()
	require.Equal(t, utils.StateRunning, code)
	require.Empty(t, reason)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, service.Stop(ctx))

	code, reason = service.State()
	require.Equal(t, utils.StateOffline, code)
	require.Empty(t, reason)
}

func TestRTNServiceProcessesAutoPullEventIntoOperation(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	host := insertTestEbicsHost(t, db, "HOST-RTN")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-RTN", "USER-RTN", true)
	account := insertTestRTNAutoPullRemoteAccount(t, db, "rtn-account")
	subscriber.RemoteAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())
	rule := insertTestRule(t, db, "rtn-download", false)
	require.NoError(t, db.Insert(&model.RuleAccess{
		RuleID:          rule.ID,
		RemoteAccountID: utils.NewNullInt64(account.ID),
	}).Run())
	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "rtn-download-profile",
		OrderType:     "BTD",
		Direction:     "DOWNLOAD",
		ServiceName:   "MCT",
		MsgName:       "camt.054",
		DefaultRuleID: utils.NewNullInt64(rule.ID),
		IsEnabled:     true,
	})
	client := insertTestRTNAutoPullClient(t, db, "ebics-rtn-client")
	t.Cleanup(func() { delete(services.Clients, client.Name) })
	insertTestActiveContractItem(t, db, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:  "ORDER_TYPE",
		ItemKey:   "BTD",
		OrderType: "BTD",
		MsgName:   "camt.054",
		IsEnabled: true,
	})
	provider := insertTestRTNProvider(t, db, subscriber.ID, "AUTO", client.ID)
	fake := newFakeRTNProvider()

	service := NewRTNService(db)
	service.providerFactory = func(cfg *model.EbicsRTNProvider) (rtn.Provider, error) {
		require.Equal(t, provider.ID, cfg.ID)
		return fake, nil
	}

	require.NoError(t, service.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})

	fake.events <- rtn.RawEvent{
		Source:        provider.Name,
		EventID:       "evt-001",
		CorrelationID: "corr-001",
		ReceivedAt:    time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		Metadata: map[string]any{
			"orderTypeHint": "BTD",
			"profileID":     "rtn-download-profile",
			"msgName":       "camt.054",
		},
	}

	require.Eventually(t, func() bool {
		count, _ := db.Count(&model.EbicsOperation{}).Where("rtn_event_id IS NOT NULL").Run()
		return count == 1
	}, 5*time.Second, 50*time.Millisecond)

	var event model.EbicsRTNEvent
	require.NoError(t, db.Get(&event, "event_id=?", "evt-001").Run())
	assert.Equal(t, "PROCESSING", event.Status)
	assert.Equal(t, 1, event.Attempts)
	assert.Zero(t, event.ProcessedAt)
	assert.Equal(t, model.EbicsOperationStatusWaitingPayloadTransferForRuntime(), event.PayloadMap["autoPullStatus"])
	assert.Equal(t, model.EbicsGatewayOutcomePendingBankForRuntime(), event.PayloadMap["autoPullOutcome"])
	assert.Equal(t, model.EbicsRetryDecisionNoRetryForRuntime(), event.PayloadMap["autoPullRetry"])

	var operation model.EbicsOperation
	require.NoError(t, db.Get(&operation, "rtn_event_id=?", event.ID).Run())
	assert.Equal(t, "BTD", operation.OrderType)
	assert.Empty(t, operation.TransactionID)
	assert.Equal(t, model.EbicsTransportModeAutoTriggeredForRuntime(), operation.TransportMode)
	assert.Equal(t, model.EbicsOperationTypePayloadForRuntime(), operation.OperationType)
	assert.Equal(t, model.EbicsOperationDirectionInboundForRuntime(), operation.Direction)
	assert.Equal(t, model.EbicsOperationStatusWaitingPayloadTransferForRuntime(), operation.Status)
	assert.EqualValues(t, event.ID, operation.RTNEventID.Int64)
	assert.Equal(t, provider.Name, operation.MetadataMap["rtnProviderName"])

	var transfer model.Transfer
	require.NoError(t, db.Get(&transfer, "id=?", operation.TransferID.Int64).Run())
	assert.Equal(t, types.StatusPlanned, transfer.Status)
	assert.Equal(t, rule.ID, transfer.RuleID)
	assert.Equal(t, client.ID, transfer.ClientID.Int64)
	assert.Equal(t, account.ID, transfer.RemoteAccountID.Int64)
	assert.Equal(t, map[string]any{
		model.FollowID: transfer.TransferInfo[model.FollowID],
	}, transfer.TransferInfo)
	assert.EqualValues(t, operation.ID, requireInt64Value(t, event.PayloadMap["autoPullOperationID"]))
	assert.EqualValues(t, transfer.ID, requireInt64Value(t, event.PayloadMap["autoPullTransferID"]))
	_, clonedOperationID := transfer.TransferInfo[transferInfoKeyEbicsOperationID]
	assert.False(t, clonedOperationID, "EBICS operation correlation must not be stored in TransferInfo")
	_, clonedEventID := transfer.TransferInfo[transferInfoKeyEbicsRTNEventID]
	assert.False(t, clonedEventID, "EBICS RTN correlation must not be stored in TransferInfo")
	_, clonedMsgName := transfer.TransferInfo["msgName"]
	assert.False(t, clonedMsgName, "raw RTN metadata should not be cloned into TransferInfo")
	_, clonedPolicy := transfer.TransferInfo["autoPullPolicy"]
	assert.False(t, clonedPolicy, "raw RTN metadata should not be cloned into TransferInfo")

	var refreshed model.EbicsRTNProvider
	require.NoError(t, db.Get(&refreshed, "id=?", provider.ID).Run())
	assert.Empty(t, refreshed.LastError)
	assert.False(t, refreshed.LastConnectionAt.IsZero())
}

func TestRTNServiceManualPolicyKeepsEventForOperators(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host := insertTestEbicsHost(t, db, "HOST-MANUAL")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-MANUAL", "USER-MANUAL", true)
	provider := insertTestRTNProvider(t, db, subscriber.ID, "MANUAL", 0)
	fake := newFakeRTNProvider()

	service := NewRTNService(db)
	service.providerFactory = func(*model.EbicsRTNProvider) (rtn.Provider, error) {
		return fake, nil
	}

	require.NoError(t, service.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})

	fake.events <- rtn.RawEvent{
		Source:        provider.Name,
		EventID:       "evt-manual",
		CorrelationID: "corr-manual",
		ReceivedAt:    time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
		Metadata: map[string]any{
			"orderTypeHint": "BTD",
		},
	}

	require.Eventually(t, func() bool {
		var event model.EbicsRTNEvent
		if err := db.Get(&event, "event_id=?", "evt-manual").Run(); err != nil {
			return false
		}
		return event.Status == "RECEIVED"
	}, 5*time.Second, 50*time.Millisecond)

	count, err := db.Count(&model.EbicsOperation{}).Where("rtn_event_id IS NOT NULL").Run()
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestRTNServiceProviderErrorsArePersisted(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	host := insertTestEbicsHost(t, db, "HOST-ERR")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-ERR", "USER-ERR", true)
	client := insertTestRTNAutoPullClient(t, db, "ebics-rtn-provider-errors")
	t.Cleanup(func() { delete(services.Clients, client.Name) })
	provider := insertTestRTNProvider(t, db, subscriber.ID, "AUTO", client.ID)
	fake := newFakeRTNProvider()

	service := NewRTNService(db)
	service.providerFactory = func(*model.EbicsRTNProvider) (rtn.Provider, error) {
		return fake, nil
	}

	require.NoError(t, service.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})

	fake.errors <- assert.AnError

	require.Eventually(t, func() bool {
		var refreshed model.EbicsRTNProvider
		if err := db.Get(&refreshed, "id=?", provider.ID).Run(); err != nil {
			return false
		}
		return refreshed.LastError == assert.AnError.Error()
	}, 5*time.Second, 50*time.Millisecond)
}

func insertTestRTNProvider(
	t *testing.T,
	db *database.DB,
	subscriberID int64,
	autoPullPolicy string,
	clientID int64,
) *model.EbicsRTNProvider {
	t.Helper()

	configuration := map[string]any{
		"endpoint": "wss://127.0.0.1/rtn",
	}
	if clientID != 0 {
		configuration["clientID"] = clientID
	}

	provider := &model.EbicsRTNProvider{
		Name:              "rtn-" + autoPullPolicy,
		Transport:         "WSS",
		Enabled:           true,
		EbicsSubscriberID: subscriberID,
		ConfigurationMap:  configuration,
		AutoPullPolicy:    autoPullPolicy,
	}
	require.NoError(t, db.Insert(provider).Run())

	return provider
}

func insertTestRTNAutoPullRemoteAccount(
	t *testing.T,
	db *database.DB,
	login string,
) *model.RemoteAccount {
	t.Helper()

	partner := &model.RemoteAgent{
		Name:     "partner-" + login,
		Protocol: EBICS,
		Address:  types.Addr("127.0.0.1", 443),
		ProtoConfig: map[string]any{
			"hostID":      "HOST-RTN",
			"endpointURL": "https://bank.example.test/ebics",
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

func insertTestRTNAutoPullClient(t *testing.T, db *database.DB, name string) *model.Client {
	t.Helper()

	client := &model.Client{
		Name:     name,
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL": "https://bank.example.test/ebics",
		},
	}
	require.NoError(t, db.Insert(client).Run())

	return client
}
