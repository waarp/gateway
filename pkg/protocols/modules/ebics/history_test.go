package ebics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestCompleteNonPayloadOperationRecordsHistory(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host, subscriber := insertHistoryTestHostAndSubscriber(t, db)

	operation := &model.EbicsOperation{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OperationType:     "REPORTING",
		OrderType:         "HVD",
		Direction:         "INBOUND",
		TransportMode:     "SYNC",
		Status:            "RUNNING",
		Severity:          "INFO",
		GatewayOutcome:    "PENDING_BANK",
		RetryDecision:     "NO_RETRY",
		RequestID:         "REQ-HIST-1",
		CorrelationID:     "CORR-HIST-1",
		StartedAt:         time.Now().UTC(),
	}
	require.NoError(t, db.Insert(operation).Run())

	client := &Client{db: db}
	require.NoError(t, client.completeNonPayloadOperation(operation, "000000", ""))

	var entries model.EbicsHistoryEntries
	require.NoError(t, db.Select(&entries).Where("operation_id=?", operation.ID).Run())
	require.Len(t, entries, 1)
	assert.Equal(t, model.EbicsHistoryTypeOperationForRuntime(), entries[0].HistoryType)
	assert.Equal(t, "HVD", entries[0].OrderType)
	assert.Equal(t, "COMPLETED", entries[0].Status)
	assert.Equal(t, "REQ-HIST-1", entries[0].RequestID)
}

func insertHistoryTestHostAndSubscriber(
	t *testing.T,
	db *database.DB,
) (*model.EbicsHost, *model.EbicsSubscriber) {
	t.Helper()

	host := &model.EbicsHost{
		Name:            "host-history-test",
		HostID:          "HOST-HISTORY-TEST",
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &model.EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "partner:user",
		PartnerID:   "PARTNER",
		UserID:      "USER",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	return host, subscriber
}
