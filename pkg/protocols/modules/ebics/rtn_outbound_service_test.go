package ebics

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestRTNOutboundServiceDispatchesQueuedNotificationOverWSS(t *testing.T) {
	db := dbtest.TestDatabase(t)

	received := make(chan []byte, 1)
	server := newTestWSServer(t, received)

	host := &model.EbicsHost{
		Name:            "HOST-OUT",
		HostID:          "HOST-OUT",
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "PARTNER-OUT:USER-OUT",
		PartnerID:   "PARTNER-OUT",
		UserID:      "USER-OUT",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	set := &model.EbicsServerReportingSet{
		Name:              "hve-bank-out",
		EbicsHostID:       host.ID,
		EbicsSubscriberID: sql.NullInt64{Int64: subscriber.ID, Valid: true},
		SourceOrderType:   "HVE",
		VersionTag:        "v1",
		Status:            "ACTIVE",
		PublishedAt:       time.Now().UTC(),
	}
	require.NoError(t, db.Insert(set).Run())

	item := &model.EbicsServerReportingItem{
		ServerReportingSetID: set.ID,
		ItemKey:              "report-1",
		OrderID:              "ORDER-1",
		ServiceName:          "MCT",
		MsgName:              "camt.054",
		IsEnabled:            true,
		ResponsePayload:      []byte("response-payload"),
		OriginalPayload:      []byte("original-payload"),
	}
	require.NoError(t, db.Insert(item).Run())

	provider := &model.EbicsRTNOutboundProvider{
		Name:              "outbound-a",
		Transport:         "WSS",
		Enabled:           true,
		EbicsSubscriberID: subscriber.ID,
		ConfigurationMap: map[string]any{
			"endpoint": server,
		},
	}
	require.NoError(t, db.Insert(provider).Run())

	notification, err := QueueRTNOutboundNotification(db, provider.ID, set, item)
	require.NoError(t, err)

	service := NewRTNOutboundService(db)
	require.NoError(t, service.dispatchDueNotificationsAt(context.Background(), time.Now().UTC()))

	select {
	case raw := <-received:
		var payload map[string]any
		require.NoError(t, json.Unmarshal(raw, &payload))
		assert.Equal(t, "REPORT_AVAILABLE", payload["eventType"])
		assert.Equal(t, "HVE", payload["orderTypeHint"])
		assert.Equal(t, "report-1", payload["serverReportingItemKey"])
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for outbound RTN notification")
	}

	var refreshed model.EbicsRTNOutboundNotification
	require.NoError(t, db.Get(&refreshed, "id=?", notification.ID).Run())
	assert.Equal(t, model.EbicsRTNOutboundNotificationStatusSentForRuntime(), refreshed.Status)
	assert.False(t, refreshed.SentAt.IsZero())

	var refreshedProvider model.EbicsRTNOutboundProvider
	require.NoError(t, db.Get(&refreshedProvider, "id=?", provider.ID).Run())
	assert.False(t, refreshedProvider.LastConnectionAt.IsZero())
	assert.Empty(t, refreshedProvider.LastError)
}

func TestRTNOutboundServiceFailsDisabledProviderWithoutSending(t *testing.T) {
	db := dbtest.TestDatabase(t)

	received := make(chan []byte, 1)
	server := newTestWSServer(t, received)

	host := &model.EbicsHost{
		Name:            "HOST-OUT-DISABLED",
		HostID:          "HOST-OUT-DISABLED",
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "PARTNER-OUT-DISABLED:USER-OUT-DISABLED",
		PartnerID:   "PARTNER-OUT-DISABLED",
		UserID:      "USER-OUT-DISABLED",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	set := &model.EbicsServerReportingSet{
		Name:              "hve-bank-out-disabled",
		EbicsHostID:       host.ID,
		EbicsSubscriberID: sql.NullInt64{Int64: subscriber.ID, Valid: true},
		SourceOrderType:   "HVE",
		VersionTag:        "v1",
		Status:            "ACTIVE",
		PublishedAt:       time.Now().UTC(),
	}
	require.NoError(t, db.Insert(set).Run())

	item := &model.EbicsServerReportingItem{
		ServerReportingSetID: set.ID,
		ItemKey:              "report-disabled-provider",
		OrderID:              "ORDER-DISABLED",
		ServiceName:          "MCT",
		MsgName:              "camt.054",
		IsEnabled:            true,
		ResponsePayload:      []byte("response-payload"),
		OriginalPayload:      []byte("original-payload"),
	}
	require.NoError(t, db.Insert(item).Run())

	provider := &model.EbicsRTNOutboundProvider{
		Name:              "outbound-disabled",
		Transport:         "WSS",
		Enabled:           false,
		EbicsSubscriberID: subscriber.ID,
		ConfigurationMap: map[string]any{
			"endpoint": server,
		},
	}
	require.NoError(t, db.Insert(provider).Run())

	notification, err := QueueRTNOutboundNotification(db, provider.ID, set, item)
	require.NoError(t, err)

	service := NewRTNOutboundService(db)
	require.NoError(t, service.dispatchDueNotificationsAt(context.Background(), time.Now().UTC()))

	select {
	case raw := <-received:
		t.Fatalf("unexpected outbound RTN notification sent: %s", string(raw))
	default:
	}

	var refreshed model.EbicsRTNOutboundNotification
	require.NoError(t, db.Get(&refreshed, "id=?", notification.ID).Run())
	assert.Equal(t, model.EbicsRTNOutboundNotificationStatusFailedForRuntime(), refreshed.Status)
	assert.Contains(t, refreshed.LastError, "disabled")
}

func newTestWSServer(t *testing.T, received chan<- []byte) string {
	t.Helper()

	srv := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		defer func() { _ = conn.Close() }()

		var raw []byte
		require.NoError(t, websocket.Message.Receive(conn, &raw))
		received <- raw
	}))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}
